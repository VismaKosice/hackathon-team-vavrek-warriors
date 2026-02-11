package mutations

import (
	"time"

	json "github.com/goccy/go-json"

	"pension-engine/internal/model"
	"pension-engine/internal/schemeregistry"
)

type projectFutureBenefitsProps struct {
	ProjectionStartDate    string `json:"projection_start_date"`
	ProjectionEndDate      string `json:"projection_end_date"`
	ProjectionIntervalMths int    `json:"projection_interval_months"`
}

type ProjectFutureBenefitsHandler struct{}

func (h *ProjectFutureBenefitsHandler) Execute(state *model.Situation, mutation *model.Mutation) ([]model.CalculationMessage, bool) {
	if state.Dossier == nil {
		return []model.CalculationMessage{{
			Level:   model.LevelCritical,
			Code:    "DOSSIER_NOT_FOUND",
			Message: "No dossier exists",
		}}, true
	}

	if len(state.Dossier.Policies) == 0 {
		return []model.CalculationMessage{{
			Level:   model.LevelCritical,
			Code:    "NO_POLICIES",
			Message: "Dossier has no policies",
		}}, true
	}

	var props projectFutureBenefitsProps
	json.Unmarshal(mutation.MutationProperties, &props)

	if props.ProjectionEndDate <= props.ProjectionStartDate {
		return []model.CalculationMessage{{
			Level:   model.LevelCritical,
			Code:    "INVALID_DATE_RANGE",
			Message: "projection_end_date must be after projection_start_date",
		}}, true
	}

	var msgs []model.CalculationMessage
	for _, p := range state.Dossier.Policies {
		if props.ProjectionStartDate < p.EmploymentStartDate {
			msgs = append(msgs, model.CalculationMessage{
				Level:   model.LevelWarning,
				Code:    "PROJECTION_BEFORE_EMPLOYMENT",
				Message: "Projection start date is before employment start date for policy " + p.PolicyID,
			})
		}
	}

	// Apply
	startDate, _ := time.Parse("2006-01-02", props.ProjectionStartDate)
	endDate, _ := time.Parse("2006-01-02", props.ProjectionEndDate)

	policies := state.Dossier.Policies
	n := len(policies)

	// Pre-parse employment start dates
	empStarts := make([]time.Time, n)
	for i, p := range policies {
		empStarts[i], _ = time.Parse("2006-01-02", p.EmploymentStartDate)
	}

	// Estimate projection count for pre-allocation
	months := (endDate.Year()-startDate.Year())*12 + int(endDate.Month()-startDate.Month())
	estCount := 1
	if props.ProjectionIntervalMths > 0 {
		estCount = months/props.ProjectionIntervalMths + 2
	}

	// Initialize projections arrays with pre-allocated capacity
	for i := range state.Dossier.Policies {
		state.Dossier.Policies[i].Projections = make([]model.Projection, 0, estCount)
	}

	// Fetch per-scheme accrual rates
	uniqueSchemes := uniqueSchemeIDs(policies)
	rates := schemeregistry.GetAccrualRates(uniqueSchemes)

	// Reuse years slice across iterations
	years := make([]float64, n)

	for projDate := startDate; !projDate.After(endDate); projDate = projDate.AddDate(0, props.ProjectionIntervalMths, 0) {
		dateStr := fastFormatDate(projDate)

		var totalYears float64
		for i := range policies {
			y := daysBetween(empStarts[i], projDate) / 365.25
			if y < 0 {
				y = 0
			}
			years[i] = y
			totalYears += y
		}

		var annualPension float64
		if totalYears > 0 {
			for i, p := range policies {
				rate := rates[p.SchemeID]
				annualPension += (p.Salary * p.PartTimeFactor) * years[i] * rate
			}
		}

		for i := range state.Dossier.Policies {
			var projected float64
			if totalYears > 0 {
				projected = annualPension * (years[i] / totalYears)
			}
			state.Dossier.Policies[i].Projections = append(state.Dossier.Policies[i].Projections, model.Projection{
				Date:             dateStr,
				ProjectedPension: projected,
			})
		}
	}

	return msgs, false
}

// fastFormatDate formats a time.Time as "YYYY-MM-DD" without time.Format overhead
func fastFormatDate(t time.Time) string {
	y, m, d := t.Date()
	var buf [10]byte
	buf[0] = byte('0' + y/1000)
	buf[1] = byte('0' + (y/100)%10)
	buf[2] = byte('0' + (y/10)%10)
	buf[3] = byte('0' + y%10)
	buf[4] = '-'
	buf[5] = byte('0' + m/10)
	buf[6] = byte('0' + m%10)
	buf[7] = '-'
	buf[8] = byte('0' + d/10)
	buf[9] = byte('0' + d%10)
	return string(buf[:])
}
