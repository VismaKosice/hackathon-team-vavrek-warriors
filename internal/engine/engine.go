package engine

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"pension-engine/internal/model"
	"pension-engine/internal/mutations"
)

func Process(req *model.CalculationRequest) *model.CalculationResponse {
	start := time.Now()

	state := &model.Situation{Dossier: nil}

	var allMessages []model.CalculationMessage
	var processedMutations []model.ProcessedMutation
	outcome := model.OutcomeSuccess
	hasCritical := false

	// Track last successfully applied mutation for end_situation
	lastMutationID := req.CalculationInstructions.Mutations[0].MutationID
	lastMutationIndex := 0
	lastActualAt := req.CalculationInstructions.Mutations[0].ActualAt
	appliedAny := false

	for i, mut := range req.CalculationInstructions.Mutations {
		handler, ok := mutations.Get(mut.MutationDefinitionName)
		if !ok {
			msg := model.CalculationMessage{
				ID:      len(allMessages),
				Level:   model.LevelCritical,
				Code:    "UNKNOWN_MUTATION",
				Message: fmt.Sprintf("Unknown mutation: %s", mut.MutationDefinitionName),
			}
			allMessages = append(allMessages, msg)
			processedMutations = append(processedMutations, model.ProcessedMutation{
				Mutation:                  mut,
				CalculationMessageIndexes: []int{msg.ID},
			})
			outcome = model.OutcomeFailure
			hasCritical = true
			break
		}

		// Validate
		validationMsgs := handler.Validate(state, &mut)
		var msgIndexes []int
		for _, vm := range validationMsgs {
			vm.ID = len(allMessages)
			allMessages = append(allMessages, vm)
			msgIndexes = append(msgIndexes, vm.ID)
			if vm.Level == model.LevelCritical {
				hasCritical = true
			}
		}

		if hasCritical {
			outcome = model.OutcomeFailure
			processedMutations = append(processedMutations, model.ProcessedMutation{
				Mutation:                  mut,
				CalculationMessageIndexes: msgIndexes,
			})
			break
		}

		// Apply
		applyMsgs := handler.Apply(state, &mut)
		for _, am := range applyMsgs {
			am.ID = len(allMessages)
			allMessages = append(allMessages, am)
			msgIndexes = append(msgIndexes, am.ID)
			if am.Level == model.LevelCritical {
				hasCritical = true
			}
		}

		processedMutations = append(processedMutations, model.ProcessedMutation{
			Mutation:                  mut,
			CalculationMessageIndexes: msgIndexes,
		})

		if hasCritical {
			outcome = model.OutcomeFailure
			break
		}

		// Track last successful mutation
		lastMutationID = mut.MutationID
		lastMutationIndex = i
		lastActualAt = mut.ActualAt
		appliedAny = true
	}

	// end_situation: if no mutation applied successfully, state is {dossier: null}
	endSituation := model.SituationEnvelope{
		MutationID:    lastMutationID,
		MutationIndex: lastMutationIndex,
		ActualAt:      lastActualAt,
		Situation:     *state,
	}

	// If critical on first mutation, end_situation has the initial state
	if hasCritical && !appliedAny {
		endSituation.Situation = model.Situation{Dossier: nil}
	}

	elapsed := time.Since(start)
	now := time.Now().UTC()

	if allMessages == nil {
		allMessages = []model.CalculationMessage{}
	}

	return &model.CalculationResponse{
		CalculationMetadata: model.CalculationMetadata{
			CalculationID:        uuid.New().String(),
			TenantID:             req.TenantID,
			CalculationStartedAt: now.Add(-elapsed).Format(time.RFC3339),
			CalculationCompletedAt: now.Format(time.RFC3339),
			CalculationDurationMs: elapsed.Milliseconds(),
			CalculationOutcome:   outcome,
		},
		CalculationResult: model.CalculationResult{
			Messages: allMessages,
			Mutations: processedMutations,
			EndSituation: endSituation,
			InitialSituation: model.InitialSituation{
				ActualAt:  req.CalculationInstructions.Mutations[0].ActualAt,
				Situation: model.Situation{Dossier: nil},
			},
		},
	}
}
