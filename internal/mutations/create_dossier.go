package mutations

import (
	"encoding/json"
	"strings"
	"time"

	"pension-engine/internal/model"
)

type createDossierProps struct {
	DossierID string `json:"dossier_id"`
	PersonID  string `json:"person_id"`
	Name      string `json:"name"`
	BirthDate string `json:"birth_date"`
}

type CreateDossierHandler struct{}

func (h *CreateDossierHandler) Validate(state *model.Situation, mutation *model.Mutation) []model.CalculationMessage {
	var msgs []model.CalculationMessage

	if state.Dossier != nil {
		msgs = append(msgs, model.CalculationMessage{
			Level:   model.LevelCritical,
			Code:    "DOSSIER_ALREADY_EXISTS",
			Message: "A dossier already exists",
		})
		return msgs
	}

	var props createDossierProps
	json.Unmarshal(mutation.MutationProperties, &props)

	if strings.TrimSpace(props.Name) == "" {
		msgs = append(msgs, model.CalculationMessage{
			Level:   model.LevelCritical,
			Code:    "INVALID_NAME",
			Message: "Name is empty or blank",
		})
		return msgs
	}

	if !isValidDate(props.BirthDate) || isFutureDate(props.BirthDate) {
		msgs = append(msgs, model.CalculationMessage{
			Level:   model.LevelCritical,
			Code:    "INVALID_BIRTH_DATE",
			Message: "Birth date is invalid or in the future",
		})
		return msgs
	}

	return msgs
}

func (h *CreateDossierHandler) Apply(state *model.Situation, mutation *model.Mutation) []model.CalculationMessage {
	var props createDossierProps
	json.Unmarshal(mutation.MutationProperties, &props)

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

	return nil
}

func isValidDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

func isFutureDate(s string) bool {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return false
	}
	return t.After(time.Now())
}
