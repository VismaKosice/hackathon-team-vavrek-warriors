package mutations

import (
	"encoding/json"
	"fmt"

	"pension-engine/internal/model"
)

type applyIndexationProps struct {
	Percentage      float64 `json:"percentage"`
	SchemeID        string  `json:"scheme_id,omitempty"`
	EffectiveBefore string  `json:"effective_before,omitempty"`
}

type ApplyIndexationHandler struct{}

func (h *ApplyIndexationHandler) Validate(state *model.Situation, mutation *model.Mutation) []model.CalculationMessage {
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

	var props applyIndexationProps
	json.Unmarshal(mutation.MutationProperties, &props)

	hasFilter := props.SchemeID != "" || props.EffectiveBefore != ""
	if hasFilter {
		matched := false
		for _, p := range state.Dossier.Policies {
			if matchesFilter(p, props) {
				matched = true
				break
			}
		}
		if !matched {
			msgs = append(msgs, model.CalculationMessage{
				Level:   model.LevelWarning,
				Code:    "NO_MATCHING_POLICIES",
				Message: "No policies match the provided filter criteria",
			})
		}
	}

	return msgs
}

func (h *ApplyIndexationHandler) Apply(state *model.Situation, mutation *model.Mutation) []model.CalculationMessage {
	var props applyIndexationProps
	json.Unmarshal(mutation.MutationProperties, &props)

	var msgs []model.CalculationMessage

	for i := range state.Dossier.Policies {
		if !matchesFilter(state.Dossier.Policies[i], props) {
			continue
		}

		newSalary := state.Dossier.Policies[i].Salary * (1 + props.Percentage)
		if newSalary < 0 {
			newSalary = 0
			msgs = append(msgs, model.CalculationMessage{
				Level:   model.LevelWarning,
				Code:    "NEGATIVE_SALARY_CLAMPED",
				Message: fmt.Sprintf("Salary for policy %s clamped to 0", state.Dossier.Policies[i].PolicyID),
			})
		}
		state.Dossier.Policies[i].Salary = newSalary
	}

	return msgs
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
