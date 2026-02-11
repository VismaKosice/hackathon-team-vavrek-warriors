package engine

import (
	"math/rand"
	"time"

	"pension-engine/internal/model"
	"pension-engine/internal/mutations"
)

func Process(req *model.CalculationRequest) *model.CalculationResponse {
	startTime := time.Now().UTC()

	state := &model.Situation{Dossier: nil}

	mutCount := len(req.CalculationInstructions.Mutations)
	allMessages := make([]model.CalculationMessage, 0, mutCount*2)
	processedMutations := make([]model.ProcessedMutation, 0, mutCount)
	outcome := model.OutcomeSuccess
	hasCritical := false

	lastMutationID := req.CalculationInstructions.Mutations[0].MutationID
	lastMutationIndex := 0
	lastActualAt := req.CalculationInstructions.Mutations[0].ActualAt
	appliedAny := false

	for i, mut := range req.CalculationInstructions.Mutations {
		handler, ok := mutations.Get(mut.MutationDefinitionName)
		if !ok {
			msgID := len(allMessages)
			allMessages = append(allMessages, model.CalculationMessage{
				ID:      msgID,
				Level:   model.LevelCritical,
				Code:    "UNKNOWN_MUTATION",
				Message: "Unknown mutation: " + mut.MutationDefinitionName,
			})
			processedMutations = append(processedMutations, model.ProcessedMutation{
				Mutation:                  mut,
				CalculationMessageIndexes: []int{msgID},
			})
			outcome = model.OutcomeFailure
			hasCritical = true
			break
		}

		msgs, critical := handler.Execute(state, &mut)
		if critical {
			hasCritical = true
		}

		var msgIndexes []int
		if len(msgs) > 0 {
			msgIndexes = make([]int, len(msgs))
			for j := range msgs {
				msgs[j].ID = len(allMessages)
				msgIndexes[j] = msgs[j].ID
				allMessages = append(allMessages, msgs[j])
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

		lastMutationID = mut.MutationID
		lastMutationIndex = i
		lastActualAt = mut.ActualAt
		appliedAny = true
	}

	endSituation := model.SituationEnvelope{
		MutationID:    lastMutationID,
		MutationIndex: lastMutationIndex,
		ActualAt:      lastActualAt,
		Situation:     *state,
	}

	if hasCritical && !appliedAny {
		endSituation.Situation = model.Situation{Dossier: nil}
	}

	endTime := time.Now().UTC()

	return &model.CalculationResponse{
		CalculationMetadata: model.CalculationMetadata{
			CalculationID:          fastUUID(),
			TenantID:               req.TenantID,
			CalculationStartedAt:   startTime.Format(time.RFC3339),
			CalculationCompletedAt: endTime.Format(time.RFC3339),
			CalculationDurationMs:  endTime.Sub(startTime).Milliseconds(),
			CalculationOutcome:     outcome,
		},
		CalculationResult: model.CalculationResult{
			Messages:  allMessages,
			Mutations: processedMutations,
			EndSituation: endSituation,
			InitialSituation: model.InitialSituation{
				ActualAt:  req.CalculationInstructions.Mutations[0].ActualAt,
				Situation: model.Situation{Dossier: nil},
			},
		},
	}
}

// fastUUID generates a UUID v4 string using math/rand instead of crypto/rand.
func fastUUID() string {
	r1 := rand.Uint64()
	r2 := rand.Uint64()
	var b [16]byte
	b[0] = byte(r1)
	b[1] = byte(r1 >> 8)
	b[2] = byte(r1 >> 16)
	b[3] = byte(r1 >> 24)
	b[4] = byte(r1 >> 32)
	b[5] = byte(r1 >> 40)
	b[6] = byte(r1 >> 48)
	b[7] = byte(r1 >> 56)
	b[8] = byte(r2)
	b[9] = byte(r2 >> 8)
	b[10] = byte(r2 >> 16)
	b[11] = byte(r2 >> 24)
	b[12] = byte(r2 >> 32)
	b[13] = byte(r2 >> 40)
	b[14] = byte(r2 >> 48)
	b[15] = byte(r2 >> 56)

	// Set version 4 and variant bits
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80

	const h = "0123456789abcdef"
	var s [36]byte
	for i, idx := 0, 0; i < 16; i++ {
		if i == 4 || i == 6 || i == 8 || i == 10 {
			s[idx] = '-'
			idx++
		}
		s[idx] = h[b[i]>>4]
		s[idx+1] = h[b[i]&0x0f]
		idx += 2
	}
	return string(s[:])
}
