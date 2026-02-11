package engine

import (
	"encoding/json"
	"testing"

	"pension-engine/internal/model"
)

func TestCreateDossier(t *testing.T) {
	req := &model.CalculationRequest{
		TenantID: "test-tenant",
		CalculationInstructions: model.CalculationInstructions{
			Mutations: []model.Mutation{
				{
					MutationID:             "a1111111-1111-1111-1111-111111111111",
					MutationDefinitionName: "create_dossier",
					MutationType:           "DOSSIER_CREATION",
					ActualAt:               "2020-01-01",
					MutationProperties: json.RawMessage(`{
						"dossier_id": "d2222222-2222-2222-2222-222222222222",
						"person_id": "p3333333-3333-3333-3333-333333333333",
						"name": "Jane Doe",
						"birth_date": "1960-06-15"
					}`),
				},
			},
		},
	}

	resp := Process(req)

	if resp.CalculationMetadata.CalculationOutcome != "SUCCESS" {
		t.Fatalf("expected SUCCESS, got %s", resp.CalculationMetadata.CalculationOutcome)
	}

	if resp.CalculationMetadata.TenantID != "test-tenant" {
		t.Fatalf("expected tenant_id test-tenant, got %s", resp.CalculationMetadata.TenantID)
	}

	if len(resp.CalculationResult.Messages) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(resp.CalculationResult.Messages))
	}

	if len(resp.CalculationResult.Mutations) != 1 {
		t.Fatalf("expected 1 mutation, got %d", len(resp.CalculationResult.Mutations))
	}

	sit := resp.CalculationResult.EndSituation.Situation
	if sit.Dossier == nil {
		t.Fatal("expected dossier to be created")
	}

	if sit.Dossier.DossierID != "d2222222-2222-2222-2222-222222222222" {
		t.Fatalf("expected dossier_id d2222222-..., got %s", sit.Dossier.DossierID)
	}

	if sit.Dossier.Status != "ACTIVE" {
		t.Fatalf("expected status ACTIVE, got %s", sit.Dossier.Status)
	}

	if len(sit.Dossier.Persons) != 1 {
		t.Fatalf("expected 1 person, got %d", len(sit.Dossier.Persons))
	}

	p := sit.Dossier.Persons[0]
	if p.Name != "Jane Doe" {
		t.Fatalf("expected name Jane Doe, got %s", p.Name)
	}

	if len(sit.Dossier.Policies) != 0 {
		t.Fatalf("expected 0 policies, got %d", len(sit.Dossier.Policies))
	}

	// initial_situation should have null dossier
	if resp.CalculationResult.InitialSituation.Situation.Dossier != nil {
		t.Fatal("expected initial situation dossier to be null")
	}

	if resp.CalculationResult.InitialSituation.ActualAt != "2020-01-01" {
		t.Fatalf("expected initial actual_at 2020-01-01, got %s", resp.CalculationResult.InitialSituation.ActualAt)
	}

	// end_situation metadata
	if resp.CalculationResult.EndSituation.MutationID != "a1111111-1111-1111-1111-111111111111" {
		t.Fatalf("unexpected end_situation mutation_id")
	}
	if resp.CalculationResult.EndSituation.MutationIndex != 0 {
		t.Fatalf("expected mutation_index 0, got %d", resp.CalculationResult.EndSituation.MutationIndex)
	}
}

func TestCreateDossierAlreadyExists(t *testing.T) {
	req := &model.CalculationRequest{
		TenantID: "test-tenant",
		CalculationInstructions: model.CalculationInstructions{
			Mutations: []model.Mutation{
				{
					MutationID:             "a1111111-1111-1111-1111-111111111111",
					MutationDefinitionName: "create_dossier",
					MutationType:           "DOSSIER_CREATION",
					ActualAt:               "2020-01-01",
					MutationProperties: json.RawMessage(`{
						"dossier_id": "d2222222-2222-2222-2222-222222222222",
						"person_id": "p3333333-3333-3333-3333-333333333333",
						"name": "Jane Doe",
						"birth_date": "1960-06-15"
					}`),
				},
				{
					MutationID:             "b4444444-4444-4444-4444-444444444444",
					MutationDefinitionName: "create_dossier",
					MutationType:           "DOSSIER_CREATION",
					ActualAt:               "2020-01-02",
					MutationProperties: json.RawMessage(`{
						"dossier_id": "d5555555-5555-5555-5555-555555555555",
						"person_id": "p6666666-6666-6666-6666-666666666666",
						"name": "John Doe",
						"birth_date": "1970-01-01"
					}`),
				},
			},
		},
	}

	resp := Process(req)

	if resp.CalculationMetadata.CalculationOutcome != "FAILURE" {
		t.Fatalf("expected FAILURE, got %s", resp.CalculationMetadata.CalculationOutcome)
	}

	if len(resp.CalculationResult.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(resp.CalculationResult.Messages))
	}

	if resp.CalculationResult.Messages[0].Code != "DOSSIER_ALREADY_EXISTS" {
		t.Fatalf("expected DOSSIER_ALREADY_EXISTS, got %s", resp.CalculationResult.Messages[0].Code)
	}

	// Should include both mutations (first succeeded, second failed)
	if len(resp.CalculationResult.Mutations) != 2 {
		t.Fatalf("expected 2 processed mutations, got %d", len(resp.CalculationResult.Mutations))
	}

	// end_situation should reflect state after first (successful) mutation
	if resp.CalculationResult.EndSituation.Situation.Dossier == nil {
		t.Fatal("expected dossier from first mutation in end_situation")
	}
	if resp.CalculationResult.EndSituation.MutationID != "a1111111-1111-1111-1111-111111111111" {
		t.Fatalf("end_situation should reference last successful mutation")
	}
}
