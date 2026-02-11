package mutations

import (
	"strconv"

	json "github.com/goccy/go-json"

	"pension-engine/internal/model"
)

type addPolicyProps struct {
	SchemeID            string  `json:"scheme_id"`
	EmploymentStartDate string  `json:"employment_start_date"`
	Salary              float64 `json:"salary"`
	PartTimeFactor      float64 `json:"part_time_factor"`
}

type AddPolicyHandler struct{}

func (h *AddPolicyHandler) Execute(state *model.Situation, mutation *model.Mutation) ([]model.CalculationMessage, bool, []byte, []byte) {
	if state.Dossier == nil {
		return []model.CalculationMessage{{
			Level:   model.LevelCritical,
			Code:    "DOSSIER_NOT_FOUND",
			Message: "No dossier exists",
		}}, true, emptyPatch, emptyPatch
	}

	var props addPolicyProps
	json.Unmarshal(mutation.MutationProperties, &props)

	if props.Salary < 0 {
		return []model.CalculationMessage{{
			Level:   model.LevelCritical,
			Code:    "INVALID_SALARY",
			Message: "Salary must be non-negative",
		}}, true, emptyPatch, emptyPatch
	}

	if props.PartTimeFactor < 0 || props.PartTimeFactor > 1 {
		return []model.CalculationMessage{{
			Level:   model.LevelCritical,
			Code:    "INVALID_PART_TIME_FACTOR",
			Message: "Part-time factor must be between 0 and 1",
		}}, true, emptyPatch, emptyPatch
	}

	var msgs []model.CalculationMessage

	// Check for duplicate policy (same scheme_id AND same employment_start_date) - WARNING only
	for _, p := range state.Dossier.Policies {
		if p.SchemeID == props.SchemeID && p.EmploymentStartDate == props.EmploymentStartDate {
			msgs = append(msgs, model.CalculationMessage{
				Level:   model.LevelWarning,
				Code:    "DUPLICATE_POLICY",
				Message: "A policy with scheme_id " + props.SchemeID + " and employment_start_date " + props.EmploymentStartDate + " already exists",
			})
			break
		}
	}

	// Apply
	state.Dossier.PolicySeq++
	policyID := state.Dossier.DossierID + "-" + strconv.Itoa(state.Dossier.PolicySeq)

	state.Dossier.Policies = append(state.Dossier.Policies, model.Policy{
		PolicyID:            policyID,
		SchemeID:            props.SchemeID,
		EmploymentStartDate: props.EmploymentStartDate,
		Salary:              props.Salary,
		PartTimeFactor:      props.PartTimeFactor,
		AttainablePension:   nil,
		Projections:         nil,
	})

	// Patches: add the new policy at the end of the array
	idx := len(state.Dossier.Policies) - 1
	path := "/dossier/policies/" + strconv.Itoa(idx)
	fwd := marshalPatches([]patchOp{{Op: "add", Path: path, Value: marshalValue(state.Dossier.Policies[idx])}})
	bwd := marshalPatches([]patchOp{{Op: "remove", Path: path}})

	return msgs, false, fwd, bwd
}
