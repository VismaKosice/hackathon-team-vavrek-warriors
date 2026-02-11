package engine

import (
	"math/rand"
	"time"

	json "github.com/goccy/go-json"

	"pension-engine/internal/jsonpatch"
	"pension-engine/internal/model"
	"pension-engine/internal/mutations"
)

var emptyPatch = []byte("[]")

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

	// Cache the generic representation: "after" of mutation i becomes "before" of mutation i+1
	lastGeneric := situationToGeneric(state)

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
				ForwardPatch:              emptyPatch,
				BackwardPatch:             emptyPatch,
				CalculationMessageIndexes: []int{msgID},
			})
			outcome = model.OutcomeFailure
			hasCritical = true
			break
		}

		beforeGeneric := lastGeneric

		msgs, critical := handler.Execute(state, &mut)
		if critical {
			hasCritical = true
		}

		// Convert after state and generate both patches in single traversal
		afterGeneric := situationToGeneric(state)
		fwdJSON, bwdJSON := generatePatches(beforeGeneric, afterGeneric)
		lastGeneric = afterGeneric

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
			ForwardPatch:              fwdJSON,
			BackwardPatch:             bwdJSON,
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

// situationToGeneric converts a Situation directly to a generic map for JSON diffing.
// Avoids the expensive json.Marshal + json.Unmarshal roundtrip.
func situationToGeneric(s *model.Situation) interface{} {
	m := make(map[string]interface{}, 1)
	if s.Dossier == nil {
		m["dossier"] = nil
		return m
	}

	d := s.Dossier
	dm := make(map[string]interface{}, 5)
	dm["dossier_id"] = d.DossierID
	dm["status"] = d.Status

	if d.RetirementDate != nil {
		dm["retirement_date"] = *d.RetirementDate
	} else {
		dm["retirement_date"] = nil
	}

	persons := make([]interface{}, len(d.Persons))
	for i, p := range d.Persons {
		persons[i] = map[string]interface{}{
			"person_id":  p.PersonID,
			"role":       p.Role,
			"name":       p.Name,
			"birth_date": p.BirthDate,
		}
	}
	dm["persons"] = persons

	policies := make([]interface{}, len(d.Policies))
	for i, p := range d.Policies {
		pm := make(map[string]interface{}, 7)
		pm["policy_id"] = p.PolicyID
		pm["scheme_id"] = p.SchemeID
		pm["employment_start_date"] = p.EmploymentStartDate
		pm["salary"] = p.Salary
		pm["part_time_factor"] = p.PartTimeFactor

		if p.AttainablePension != nil {
			pm["attainable_pension"] = *p.AttainablePension
		} else {
			pm["attainable_pension"] = nil
		}

		if p.Projections != nil {
			projs := make([]interface{}, len(p.Projections))
			for j, proj := range p.Projections {
				projs[j] = map[string]interface{}{
					"date":             proj.Date,
					"projected_pension": proj.ProjectedPension,
				}
			}
			pm["projections"] = projs
		} else {
			pm["projections"] = nil
		}
		policies[i] = pm
	}
	dm["policies"] = policies

	m["dossier"] = dm
	return m
}

// generatePatches produces forward and backward RFC 6902 JSON Patch documents in a single traversal.
func generatePatches(before, after interface{}) (fwd, bwd []byte) {
	fwdOps, bwdOps := jsonpatch.DiffBoth(before, after, "")

	if len(fwdOps) == 0 {
		fwd = emptyPatch
	} else {
		fwd, _ = json.Marshal(fwdOps)
	}

	if len(bwdOps) == 0 {
		bwd = emptyPatch
	} else {
		bwd, _ = json.Marshal(bwdOps)
	}

	return fwd, bwd
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
