package mutations

import (
	"strings"
	"time"

	json "github.com/goccy/go-json"

	"pension-engine/internal/model"
)

type createDossierProps struct {
	DossierID string `json:"dossier_id"`
	PersonID  string `json:"person_id"`
	Name      string `json:"name"`
	BirthDate string `json:"birth_date"`
}

type CreateDossierHandler struct{}

func (h *CreateDossierHandler) Execute(state *model.Situation, mutation *model.Mutation) ([]model.CalculationMessage, bool, []byte, []byte) {
	if state.Dossier != nil {
		return []model.CalculationMessage{{
			Level:   model.LevelCritical,
			Code:    "DOSSIER_ALREADY_EXISTS",
			Message: "A dossier already exists",
		}}, true, emptyPatch, emptyPatch
	}

	var props createDossierProps
	json.Unmarshal(mutation.MutationProperties, &props)

	if strings.TrimSpace(props.Name) == "" {
		return []model.CalculationMessage{{
			Level:   model.LevelCritical,
			Code:    "INVALID_NAME",
			Message: "Name is empty or blank",
		}}, true, emptyPatch, emptyPatch
	}

	// Single parse: validate date and check future in one operation
	t, err := time.Parse("2006-01-02", props.BirthDate)
	if err != nil || t.After(time.Now()) {
		return []model.CalculationMessage{{
			Level:   model.LevelCritical,
			Code:    "INVALID_BIRTH_DATE",
			Message: "Birth date is invalid or in the future",
		}}, true, emptyPatch, emptyPatch
	}

	// Apply
	state.Dossier = &model.Dossier{
		DossierID:      props.DossierID,
		Status:         "ACTIVE",
		RetirementDate: nil,
		Persons: []model.Person{
			{
				PersonID:  props.PersonID,
				Role:      "PARTICIPANT",
				Name:      props.Name,
				BirthDate: props.BirthDate,
			},
		},
		Policies:  []model.Policy{},
		PolicySeq: 0,
	}

	// Patches: /dossier goes from null to the new dossier
	fwd := marshalPatches([]patchOp{{Op: "replace", Path: "/dossier", Value: marshalValue(state.Dossier)}})
	bwd := marshalPatches([]patchOp{{Op: "replace", Path: "/dossier", Value: jsonNull}})

	return nil, false, fwd, bwd
}
