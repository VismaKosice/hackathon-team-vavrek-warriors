package mutations

import (
	"fmt"
	"strconv"
	"time"

	json "github.com/goccy/go-json"

	"pension-engine/internal/model"
	"pension-engine/internal/schemeregistry"
)

type calcRetirementProps struct {
	RetirementDate string `json:"retirement_date"`
}

type CalculateRetirementBenefitHandler struct{}

func (h *CalculateRetirementBenefitHandler) Execute(state *model.Situation, mutation *model.Mutation) ([]model.CalculationMessage, bool, []byte, []byte) {
	if state.Dossier == nil {
		return []model.CalculationMessage{{
			Level:   model.LevelCritical,
			Code:    "DOSSIER_NOT_FOUND",
			Message: "No dossier exists",
		}}, true, emptyPatch, emptyPatch
	}

	if len(state.Dossier.Policies) == 0 {
		return []model.CalculationMessage{{
			Level:   model.LevelCritical,
			Code:    "NO_POLICIES",
			Message: "Dossier has no policies",
		}}, true, emptyPatch, emptyPatch
	}

	var props calcRetirementProps
	json.Unmarshal(mutation.MutationProperties, &props)

	retDate, _ := fastParseDate(props.RetirementDate)
	birthDate, _ := fastParseDate(state.Dossier.Persons[0].BirthDate)

	policies := state.Dossier.Policies
	n := len(policies)

	// Single pass: parse dates, compute years of service and effective salaries
	age := calendarYears(birthDate, retDate)
	years := make([]float64, n)
	effectiveSalaries := make([]float64, n)
	var totalYears float64

	for i, p := range policies {
		empStart, _ := fastParseDate(p.EmploymentStartDate)
		y := daysBetween(empStart, retDate) / 365.25
		if y < 0 {
			y = 0
		}
		years[i] = y
		effectiveSalaries[i] = p.Salary * p.PartTimeFactor
		totalYears += y
	}

	// Eligibility: must be >= 65 years old OR total years of service >= 40
	if age < 65 && totalYears < 40 {
		return []model.CalculationMessage{{
			Level:   model.LevelCritical,
			Code:    "NOT_ELIGIBLE",
			Message: fmt.Sprintf("Participant is %d years old with %.1f years of service", int(age), totalYears),
		}}, true, emptyPatch, emptyPatch
	}

	// Check retirement before employment (WARNING per violating policy)
	var msgs []model.CalculationMessage
	for _, p := range policies {
		if props.RetirementDate < p.EmploymentStartDate {
			msgs = append(msgs, model.CalculationMessage{
				Level:   model.LevelWarning,
				Code:    "RETIREMENT_BEFORE_EMPLOYMENT",
				Message: "Retirement date is before employment start date for policy " + p.PolicyID,
			})
		}
	}

	// Capture old state for backward patches
	oldStatus := state.Dossier.Status

	// Fetch per-scheme accrual rates
	uniqueSchemes := uniqueSchemeIDs(policies)
	rates := schemeregistry.GetAccrualRates(uniqueSchemes)

	// Compute annual pension with per-scheme accrual rates
	var annualPension float64
	if totalYears > 0 {
		for i := range policies {
			rate := rates[policies[i].SchemeID]
			annualPension += effectiveSalaries[i] * years[i] * rate
		}
	}

	for i := range state.Dossier.Policies {
		var policyPension float64
		if totalYears > 0 {
			policyPension = annualPension * (years[i] / totalYears)
		}
		state.Dossier.Policies[i].AttainablePension = &policyPension
	}

	state.Dossier.Status = "RETIRED"
	state.Dossier.RetirementDate = &props.RetirementDate

	// Generate patches for: status, retirement_date, attainable_pension per policy
	fwdOps := make([]patchOp, 0, 2+n)
	bwdOps := make([]patchOp, 0, 2+n)

	fwdOps = append(fwdOps, patchOp{Op: "replace", Path: "/dossier/status", Value: marshalValue("RETIRED")})
	bwdOps = append(bwdOps, patchOp{Op: "replace", Path: "/dossier/status", Value: marshalValue(oldStatus)})

	fwdOps = append(fwdOps, patchOp{Op: "replace", Path: "/dossier/retirement_date", Value: marshalValue(props.RetirementDate)})
	bwdOps = append(bwdOps, patchOp{Op: "replace", Path: "/dossier/retirement_date", Value: jsonNull})

	for i := range state.Dossier.Policies {
		path := "/dossier/policies/" + strconv.Itoa(i) + "/attainable_pension"
		fwdOps = append(fwdOps, patchOp{Op: "replace", Path: path, Value: marshalValue(state.Dossier.Policies[i].AttainablePension)})
		bwdOps = append(bwdOps, patchOp{Op: "replace", Path: path, Value: jsonNull})
	}

	return msgs, false, marshalPatches(fwdOps), marshalPatches(bwdOps)
}

func uniqueSchemeIDs(policies []model.Policy) []string {
	seen := make(map[string]struct{}, len(policies))
	result := make([]string, 0, len(policies))
	for _, p := range policies {
		if _, ok := seen[p.SchemeID]; !ok {
			seen[p.SchemeID] = struct{}{}
			result = append(result, p.SchemeID)
		}
	}
	return result
}

func daysBetween(start, end time.Time) float64 {
	return end.Sub(start).Hours() / 24
}

func calendarYears(birth, target time.Time) float64 {
	years := float64(target.Year() - birth.Year())
	if target.Month() < birth.Month() ||
		(target.Month() == birth.Month() && target.Day() < birth.Day()) {
		years--
	}
	return years
}
