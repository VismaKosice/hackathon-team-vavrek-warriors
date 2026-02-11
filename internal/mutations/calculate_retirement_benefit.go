package mutations

import (
	"encoding/json"
	"fmt"
	"time"

	"pension-engine/internal/model"
)

type calcRetirementProps struct {
	RetirementDate string `json:"retirement_date"`
}

type CalculateRetirementBenefitHandler struct{}

func (h *CalculateRetirementBenefitHandler) Validate(state *model.Situation, mutation *model.Mutation) []model.CalculationMessage {
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

	var props calcRetirementProps
	json.Unmarshal(mutation.MutationProperties, &props)

	retDate, _ := time.Parse("2006-01-02", props.RetirementDate)
	birthDate, _ := time.Parse("2006-01-02", state.Dossier.Persons[0].BirthDate)

	// Age on retirement date (calendar years for eligibility)
	age := calendarYears(birthDate, retDate)

	// Total years of service across all policies (days / 365.25 per spec)
	var totalYears float64
	for _, p := range state.Dossier.Policies {
		empStart, _ := time.Parse("2006-01-02", p.EmploymentStartDate)
		years := daysBetween(empStart, retDate) / 365.25
		if years < 0 {
			years = 0
		}
		totalYears += years
	}

	// Eligibility: must be >= 65 years old OR total years of service >= 40
	if age < 65 && totalYears < 40 {
		msgs = append(msgs, model.CalculationMessage{
			Level:   model.LevelCritical,
			Code:    "NOT_ELIGIBLE",
			Message: fmt.Sprintf("Participant is %d years old with %.1f years of service", int(age), totalYears),
		})
		return msgs
	}

	// Check retirement before employment (WARNING per violating policy)
	for _, p := range state.Dossier.Policies {
		if props.RetirementDate < p.EmploymentStartDate {
			msgs = append(msgs, model.CalculationMessage{
				Level:   model.LevelWarning,
				Code:    "RETIREMENT_BEFORE_EMPLOYMENT",
				Message: fmt.Sprintf("Retirement date is before employment start date for policy %s", p.PolicyID),
			})
		}
	}

	return msgs
}

func (h *CalculateRetirementBenefitHandler) Apply(state *model.Situation, mutation *model.Mutation) []model.CalculationMessage {
	var props calcRetirementProps
	json.Unmarshal(mutation.MutationProperties, &props)

	retDate, _ := time.Parse("2006-01-02", props.RetirementDate)

	const accrualRate = 0.02

	policies := state.Dossier.Policies
	n := len(policies)

	// Per-policy: years of service and effective salary
	years := make([]float64, n)
	effectiveSalaries := make([]float64, n)
	var totalYears float64

	for i, p := range policies {
		empStart, _ := time.Parse("2006-01-02", p.EmploymentStartDate)
		y := daysBetween(empStart, retDate) / 365.25
		if y < 0 {
			y = 0
		}
		years[i] = y
		effectiveSalaries[i] = p.Salary * p.PartTimeFactor
		totalYears += y
	}

	// Weighted average salary
	var weightedSum float64
	for i := range policies {
		weightedSum += effectiveSalaries[i] * years[i]
	}

	var weightedAvg float64
	if totalYears > 0 {
		weightedAvg = weightedSum / totalYears
	}

	// Annual pension
	annualPension := weightedAvg * totalYears * accrualRate

	// Distribute proportionally by years of service
	for i := range state.Dossier.Policies {
		var policyPension float64
		if totalYears > 0 {
			policyPension = annualPension * (years[i] / totalYears)
		}
		state.Dossier.Policies[i].AttainablePension = &policyPension
	}

	// Update dossier status
	state.Dossier.Status = "RETIRED"
	state.Dossier.RetirementDate = &props.RetirementDate

	return nil
}

func daysBetween(start, end time.Time) float64 {
	return end.Sub(start).Hours() / 24
}

// calendarYears returns whole years between two dates (for age eligibility).
func calendarYears(birth, target time.Time) float64 {
	years := float64(target.Year() - birth.Year())
	if target.Month() < birth.Month() ||
		(target.Month() == birth.Month() && target.Day() < birth.Day()) {
		years--
	}
	return years
}
