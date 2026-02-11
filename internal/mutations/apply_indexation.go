package mutations

import (
	"strconv"

	json "github.com/goccy/go-json"

	"pension-engine/internal/model"
)

type applyIndexationProps struct {
	Percentage      float64 `json:"percentage"`
	SchemeID        string  `json:"scheme_id,omitempty"`
	EffectiveBefore string  `json:"effective_before,omitempty"`
}

type ApplyIndexationHandler struct{}

func (h *ApplyIndexationHandler) Execute(state *model.Situation, mutation *model.Mutation) ([]model.CalculationMessage, bool, []byte, []byte) {
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

	var props applyIndexationProps
	json.Unmarshal(mutation.MutationProperties, &props)

	var msgs []model.CalculationMessage
	hasFilter := props.SchemeID != "" || props.EffectiveBefore != ""

	var fwdOps, bwdOps []patchOp

	// Single pass: validate filter match AND apply indexation
	matched := false
	for i := range state.Dossier.Policies {
		if !matchesFilter(state.Dossier.Policies[i], props) {
			continue
		}
		matched = true
		oldSalary := state.Dossier.Policies[i].Salary
		newSalary := oldSalary * (1 + props.Percentage)
		if newSalary < 0 {
			newSalary = 0
			msgs = append(msgs, model.CalculationMessage{
				Level:   model.LevelWarning,
				Code:    "NEGATIVE_SALARY_CLAMPED",
				Message: "Salary for policy " + state.Dossier.Policies[i].PolicyID + " clamped to 0",
			})
		}
		state.Dossier.Policies[i].Salary = newSalary

		path := "/dossier/policies/" + strconv.Itoa(i) + "/salary"
		fwdOps = append(fwdOps, patchOp{Op: "replace", Path: path, Value: marshalValue(newSalary)})
		bwdOps = append(bwdOps, patchOp{Op: "replace", Path: path, Value: marshalValue(oldSalary)})
	}

	if hasFilter && !matched {
		msgs = append([]model.CalculationMessage{{
			Level:   model.LevelWarning,
			Code:    "NO_MATCHING_POLICIES",
			Message: "No policies match the provided filter criteria",
		}}, msgs...)
	}

	return msgs, false, marshalPatches(fwdOps), marshalPatches(bwdOps)
}

func matchesFilter(p model.Policy, props applyIndexationProps) bool {
	if props.SchemeID != "" && p.SchemeID != props.SchemeID {
		return false
	}
	if props.EffectiveBefore != "" && p.EmploymentStartDate >= props.EffectiveBefore {
		return false
	}
	return true
}
