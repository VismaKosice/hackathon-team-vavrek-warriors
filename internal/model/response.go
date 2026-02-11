package model

import "encoding/json"

type CalculationResponse struct {
	CalculationMetadata CalculationMetadata `json:"calculation_metadata"`
	CalculationResult   CalculationResult   `json:"calculation_result"`
}

type CalculationMetadata struct {
	CalculationID        string `json:"calculation_id"`
	TenantID             string `json:"tenant_id"`
	CalculationStartedAt string `json:"calculation_started_at"`
	CalculationCompletedAt string `json:"calculation_completed_at"`
	CalculationDurationMs int64  `json:"calculation_duration_ms"`
	CalculationOutcome   string `json:"calculation_outcome"`
}

type CalculationResult struct {
	Messages         []CalculationMessage `json:"messages"`
	Mutations        []ProcessedMutation  `json:"mutations"`
	EndSituation     SituationEnvelope    `json:"end_situation"`
	InitialSituation InitialSituation     `json:"initial_situation"`
}

type ProcessedMutation struct {
	Mutation                  Mutation         `json:"mutation"`
	ForwardPatch              json.RawMessage  `json:"forward_patch_to_situation_after_this_mutation"`
	BackwardPatch             json.RawMessage  `json:"backward_patch_to_previous_situation"`
	CalculationMessageIndexes []int            `json:"calculation_message_indexes,omitempty"`
}

type SituationEnvelope struct {
	MutationID    string    `json:"mutation_id"`
	MutationIndex int       `json:"mutation_index"`
	ActualAt      string    `json:"actual_at"`
	Situation     Situation `json:"situation"`
}

type InitialSituation struct {
	ActualAt  string    `json:"actual_at"`
	Situation Situation `json:"situation"`
}

type ErrorResponse struct {
	Status  int    `json:"status"`
	Message string `json:"message"`
}

const (
	OutcomeSuccess = "SUCCESS"
	OutcomeFailure = "FAILURE"
)
