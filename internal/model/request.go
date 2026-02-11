package model

import "encoding/json"

type CalculationRequest struct {
	TenantID               string                  `json:"tenant_id"`
	CalculationInstructions CalculationInstructions `json:"calculation_instructions"`
}

type CalculationInstructions struct {
	Mutations []Mutation `json:"mutations"`
}

type Mutation struct {
	MutationID             string          `json:"mutation_id"`
	MutationDefinitionName string          `json:"mutation_definition_name"`
	MutationType           string          `json:"mutation_type"`
	ActualAt               string          `json:"actual_at"`
	DossierID              string          `json:"dossier_id,omitempty"`
	MutationProperties     json.RawMessage `json:"mutation_properties"`
}
