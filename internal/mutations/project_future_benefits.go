package mutations

import (
	"encoding/json"
	"fmt"
	"time"

	"pension-engine/internal/model"
)

type projectFutureBenefitsProps struct {
	ProjectionStartDate    string `json:"projection_start_date"`
	ProjectionEndDate      string `json:"projection_end_date"`
	ProjectionIntervalMths int    `json:"projection_interval_months"`
}

type ProjectFutureBenefitsHandler struct{}

func (h *ProjectFutureBenefitsHandler) Validate(state *model.Situation, mutation *model.Mutation) []model.CalculationMessage {
	var msgs []model.CalculationMessage

	if state.Dossier == nil {
		msgs = append(msgs, model.CalculationMessage{
			Level:   model.LevelCritical,
			Code:    "DOSSIER_NOT_FOUND",
			Message: "No dossier exists",
		})
		return msgs
	}

	if len(state.Dossier.Policies) == 0 {
		msgs = append(msgs, model.CalculationMessage{
			Level:   model.LevelCritical,
			Code:    "NO_POLICIES",
			Message: "Dossier has no policies",
		})
		return msgs
	}

	var props projectFutureBenefitsProps
	json.Unmarshal(mutation.MutationProperties, &props)

	if props.ProjectionEndDate <= props.ProjectionStartDate {
		msgs = append(msgs, model.CalculationMessage{
			Level:   model.LevelCritical,
			Code:    "INVALID_DATE_RANGE",
			Message: "projection_end_date must be after projection_start_date",
		})
		return msgs
	}

	for _, p := range state.Dossier.Policies {
		if props.ProjectionStartDate < p.EmploymentStartDate {
			msgs = append(msgs, model.CalculationMessage{
				Level:   model.LevelWarning,
				Code:    "PROJECTION_BEFORE_EMPLOYMENT",
				Message: fmt.Sprintf("Projection start date is before employment start date for policy %s", p.PolicyID),
			})
		}
	}

	return msgs
}

func (h *ProjectFutureBenefitsHandler) Apply(state *model.Situation, mutation *model.Mutation) []model.CalculationMessage {
	var props projectFutureBenefitsProps
	json.Unmarshal(mutation.MutationProperties, &props)

	startDate, _ := time.Parse("2006-01-02", props.ProjectionStartDate)
	endDate, _ := time.Parse("2006-01-02", props.ProjectionEndDate)

	policies := state.Dossier.Policies
	n := len(policies)

	// Pre-parse employment start dates
	empStarts := make([]time.Time, n)
	for i, p := range policies {
		empStarts[i], _ = time.Parse("2006-01-02", p.EmploymentStartDate)
	}

	// Initialize projections arrays
	for i := range state.Dossier.Policies {
		state.Dossier.Policies[i].Projections = []model.Projection{}
	}

	const accrualRate = 0.02

	// Step through projection dates
	for projDate := startDate; !projDate.After(endDate); projDate = projDate.AddDate(0, props.ProjectionIntervalMths, 0) {
		dateStr := projDate.Format("2006-01-02")

		// Calculate years of service and effective salaries
		years := make([]float64, n)
		var totalYears float64

		for i, p := range policies {
			y := daysBetween(empStarts[i], projDate) / 365.25
			if y < 0 {
				y = 0
			}
			years[i] = y
			_ = p // use p for nothing else; salary/ptf accessed via policies[i]
			totalYears += y
		}

		// Weighted sum of effective salaries
		var weightedSum float64
		for i, p := range policies {
			weightedSum += (p.Salary * p.PartTimeFactor) * years[i]
		}

		// Annual pension
		var annualPension float64
		if totalYears > 0 {
			annualPension = weightedSum * accrualRate
		}

		// Per-policy projected pension
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

	return nil
}
