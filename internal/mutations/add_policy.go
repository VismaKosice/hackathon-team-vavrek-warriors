package mutations

import (
	"encoding/json"
	"fmt"

	"pension-engine/internal/model"
)

type addPolicyProps struct {
	SchemeID            string  `json:"scheme_id"`
	EmploymentStartDate string  `json:"employment_start_date"`
	Salary              float64 `json:"salary"`
	PartTimeFactor      float64 `json:"part_time_factor"`
}

type AddPolicyHandler struct{}

func (h *AddPolicyHandler) Validate(state *model.Situation, mutation *model.Mutation) []model.CalculationMessage {
	var msgs []model.CalculationMessage

	if state.Dossier == nil {
		msgs = append(msgs, model.CalculationMessage{
			Level:   model.LevelCritical,
			Code:    "DOSSIER_NOT_FOUND",
			Message: "No dossier exists",
		})
		return msgs
	}

	var props addPolicyProps
	json.Unmarshal(mutation.MutationProperties, &props)

	if props.Salary < 0 {
		msgs = append(msgs, model.CalculationMessage{
			Level:   model.LevelCritical,
			Code:    "INVALID_SALARY",
			Message: "Salary must be non-negative",
		})
		return msgs
	}

	if props.PartTimeFactor < 0 || props.PartTimeFactor > 1 {
		msgs = append(msgs, model.CalculationMessage{
			Level:   model.LevelCritical,
			Code:    "INVALID_PART_TIME_FACTOR",
			Message: "Part-time factor must be between 0 and 1",
		})
		return msgs
	}

	// Check for duplicate policy (same scheme_id AND same employment_start_date) — WARNING only
	for _, p := range state.Dossier.Policies {
		if p.SchemeID == props.SchemeID && p.EmploymentStartDate == props.EmploymentStartDate {
			msgs = append(msgs, model.CalculationMessage{
				Level:   model.LevelWarning,
				Code:    "DUPLICATE_POLICY",
				Message: fmt.Sprintf("A policy with scheme_id %s and employment_start_date %s already exists", props.SchemeID, props.EmploymentStartDate),
			})
			break
		}
	}

	return msgs
}

func (h *AddPolicyHandler) Apply(state *model.Situation, mutation *model.Mutation) []model.CalculationMessage {
	var props addPolicyProps
	json.Unmarshal(mutation.MutationProperties, &props)

	state.Dossier.PolicySeq++
	policyID := fmt.Sprintf("%s-%d", state.Dossier.DossierID, state.Dossier.PolicySeq)

	state.Dossier.Policies = append(state.Dossier.Policies, model.Policy{
		PolicyID:            policyID,
		SchemeID:            props.SchemeID,
		EmploymentStartDate: props.EmploymentStartDate,
		Salary:              props.Salary,
		PartTimeFactor:      props.PartTimeFactor,
		AttainablePension:   nil,
		Projections:         nil,
	})

	return nil
}
