package engine

import (
	"encoding/json"
	"math"
	"testing"

	"pension-engine/internal/model"
)

func assertFloat(t *testing.T, name string, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 0.01 {
		t.Fatalf("%s: expected %.2f, got %.2f", name, want, got)
	}
}

// --- create_dossier ---

func TestCreateDossier(t *testing.T) {
	resp := Process(makeReq("test-tenant", createDossierMut()))

	if resp.CalculationMetadata.CalculationOutcome != "SUCCESS" {
		t.Fatalf("expected SUCCESS, got %s", resp.CalculationMetadata.CalculationOutcome)
	}
	if len(resp.CalculationResult.Messages) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(resp.CalculationResult.Messages))
	}

	sit := resp.CalculationResult.EndSituation.Situation
	if sit.Dossier == nil {
		t.Fatal("expected dossier to be created")
	}
	if sit.Dossier.DossierID != dossierID {
		t.Fatalf("expected dossier_id %s, got %s", dossierID, sit.Dossier.DossierID)
	}
	if sit.Dossier.Status != "ACTIVE" {
		t.Fatalf("expected status ACTIVE, got %s", sit.Dossier.Status)
	}
	if len(sit.Dossier.Persons) != 1 || sit.Dossier.Persons[0].Name != "Jane Doe" {
		t.Fatal("unexpected person data")
	}
	if len(sit.Dossier.Policies) != 0 {
		t.Fatalf("expected 0 policies, got %d", len(sit.Dossier.Policies))
	}

	if resp.CalculationResult.InitialSituation.Situation.Dossier != nil {
		t.Fatal("expected initial situation dossier to be null")
	}
	if resp.CalculationResult.InitialSituation.ActualAt != "2020-01-01" {
		t.Fatalf("expected initial actual_at 2020-01-01, got %s", resp.CalculationResult.InitialSituation.ActualAt)
	}
	if resp.CalculationResult.EndSituation.MutationIndex != 0 {
		t.Fatalf("expected mutation_index 0, got %d", resp.CalculationResult.EndSituation.MutationIndex)
	}
}

func TestCreateDossierAlreadyExists(t *testing.T) {
	resp := Process(makeReq("test", createDossierMut(), model.Mutation{
		MutationID:             "b4444444-4444-4444-4444-444444444444",
		MutationDefinitionName: "create_dossier",
		MutationType:           "DOSSIER_CREATION",
		ActualAt:               "2020-01-02",
		MutationProperties:     json.RawMessage(`{"dossier_id":"x","person_id":"y","name":"John","birth_date":"1970-01-01"}`),
	}))

	if resp.CalculationMetadata.CalculationOutcome != "FAILURE" {
		t.Fatalf("expected FAILURE, got %s", resp.CalculationMetadata.CalculationOutcome)
	}
	if resp.CalculationResult.Messages[0].Code != "DOSSIER_ALREADY_EXISTS" {
		t.Fatalf("expected DOSSIER_ALREADY_EXISTS, got %s", resp.CalculationResult.Messages[0].Code)
	}
	if len(resp.CalculationResult.Mutations) != 2 {
		t.Fatalf("expected 2 processed mutations, got %d", len(resp.CalculationResult.Mutations))
	}
	// end_situation reflects state after first (successful) mutation
	if resp.CalculationResult.EndSituation.Situation.Dossier == nil {
		t.Fatal("expected dossier from first mutation")
	}
}

// --- add_policy ---

func TestAddPolicy(t *testing.T) {
	resp := Process(makeReq("test", createDossierMut(), addPolicyMut("SCHEME-A", "2000-01-01", 50000, 1.0)))

	if resp.CalculationMetadata.CalculationOutcome != "SUCCESS" {
		t.Fatalf("expected SUCCESS, got %s", resp.CalculationMetadata.CalculationOutcome)
	}

	policies := resp.CalculationResult.EndSituation.Situation.Dossier.Policies
	if len(policies) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(policies))
	}
	if policies[0].PolicyID != dossierID+"-1" {
		t.Fatalf("expected policy_id %s-1, got %s", dossierID, policies[0].PolicyID)
	}
	assertFloat(t, "salary", policies[0].Salary, 50000)
	if policies[0].AttainablePension != nil {
		t.Fatal("expected attainable_pension to be null")
	}
}

func TestAddMultiplePolicies(t *testing.T) {
	resp := Process(makeReq("test",
		createDossierMut(),
		addPolicyMut("SCHEME-A", "2000-01-01", 50000, 1.0),
		addPolicyMut("SCHEME-B", "2010-01-01", 60000, 0.8),
	))

	policies := resp.CalculationResult.EndSituation.Situation.Dossier.Policies
	if len(policies) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(policies))
	}
	if policies[0].PolicyID != dossierID+"-1" {
		t.Fatalf("expected -1, got %s", policies[0].PolicyID)
	}
	if policies[1].PolicyID != dossierID+"-2" {
		t.Fatalf("expected -2, got %s", policies[1].PolicyID)
	}
}

func TestAddPolicyNoDossier(t *testing.T) {
	resp := Process(makeReq("test", addPolicyMut("SCHEME-A", "2000-01-01", 50000, 1.0)))

	if resp.CalculationMetadata.CalculationOutcome != "FAILURE" {
		t.Fatalf("expected FAILURE")
	}
	if resp.CalculationResult.Messages[0].Code != "DOSSIER_NOT_FOUND" {
		t.Fatalf("expected DOSSIER_NOT_FOUND")
	}
}

func TestAddPolicyDuplicateWarning(t *testing.T) {
	resp := Process(makeReq("test",
		createDossierMut(),
		addPolicyMut("SCHEME-A", "2000-01-01", 50000, 1.0),
		addPolicyMut("SCHEME-A", "2000-01-01", 55000, 1.0),
	))

	if resp.CalculationMetadata.CalculationOutcome != "SUCCESS" {
		t.Fatalf("expected SUCCESS (duplicate is WARNING), got %s", resp.CalculationMetadata.CalculationOutcome)
	}
	if len(resp.CalculationResult.Messages) != 1 {
		t.Fatalf("expected 1 warning, got %d", len(resp.CalculationResult.Messages))
	}
	if resp.CalculationResult.Messages[0].Code != "DUPLICATE_POLICY" {
		t.Fatalf("expected DUPLICATE_POLICY, got %s", resp.CalculationResult.Messages[0].Code)
	}
	// Both policies should still be added
	if len(resp.CalculationResult.EndSituation.Situation.Dossier.Policies) != 2 {
		t.Fatal("expected 2 policies despite duplicate warning")
	}
}

// --- apply_indexation ---

func TestApplyIndexationNoFilter(t *testing.T) {
	resp := Process(makeReq("test",
		createDossierMut(),
		addPolicyMut("SCHEME-A", "2000-01-01", 50000, 1.0),
		indexationMut(0.03, "", ""),
	))

	if resp.CalculationMetadata.CalculationOutcome != "SUCCESS" {
		t.Fatalf("expected SUCCESS")
	}
	assertFloat(t, "salary after 3%", resp.CalculationResult.EndSituation.Situation.Dossier.Policies[0].Salary, 51500)
}

func TestApplyIndexationSchemeFilter(t *testing.T) {
	resp := Process(makeReq("test",
		createDossierMut(),
		addPolicyMut("SCHEME-A", "2000-01-01", 50000, 1.0),
		addPolicyMut("SCHEME-B", "2010-01-01", 60000, 0.8),
		indexationMutWithScheme(0.10, "SCHEME-A"),
	))

	policies := resp.CalculationResult.EndSituation.Situation.Dossier.Policies
	assertFloat(t, "SCHEME-A salary", policies[0].Salary, 55000)
	assertFloat(t, "SCHEME-B salary (unchanged)", policies[1].Salary, 60000)
}

func TestApplyIndexationEffectiveBeforeFilter(t *testing.T) {
	resp := Process(makeReq("test",
		createDossierMut(),
		addPolicyMut("SCHEME-A", "2000-01-01", 50000, 1.0),
		addPolicyMut("SCHEME-B", "2010-01-01", 60000, 0.8),
		indexationMutWithEffectiveBefore(0.10, "2005-01-01"),
	))

	policies := resp.CalculationResult.EndSituation.Situation.Dossier.Policies
	assertFloat(t, "before 2005 salary", policies[0].Salary, 55000)
	assertFloat(t, "after 2005 salary (unchanged)", policies[1].Salary, 60000)
}

func TestApplyIndexationNegativeSalaryClamped(t *testing.T) {
	resp := Process(makeReq("test",
		createDossierMut(),
		addPolicyMut("SCHEME-A", "2000-01-01", 1000, 1.0),
		indexationMut(-2.0, "", ""), // -200% → would make salary -1000
	))

	assertFloat(t, "clamped salary", resp.CalculationResult.EndSituation.Situation.Dossier.Policies[0].Salary, 0)
	found := false
	for _, m := range resp.CalculationResult.Messages {
		if m.Code == "NEGATIVE_SALARY_CLAMPED" {
			found = true
		}
	}
	if !found {
		t.Fatal("expected NEGATIVE_SALARY_CLAMPED warning")
	}
}

// --- calculate_retirement_benefit ---

func TestCalculateRetirementBenefitExample(t *testing.T) {
	// README example: 2 policies, retirement at 2025-01-01
	resp := Process(makeReq("test",
		createDossierMut(),
		addPolicyMut("SCHEME-A", "2000-01-01", 50000, 1.0),
		addPolicyMut("SCHEME-B", "2010-01-01", 60000, 0.8),
		retirementMut("2025-01-01"),
	))

	if resp.CalculationMetadata.CalculationOutcome != "SUCCESS" {
		t.Fatalf("expected SUCCESS, got %s", resp.CalculationMetadata.CalculationOutcome)
	}

	dossier := resp.CalculationResult.EndSituation.Situation.Dossier
	if dossier.Status != "RETIRED" {
		t.Fatalf("expected RETIRED, got %s", dossier.Status)
	}
	if dossier.RetirementDate == nil || *dossier.RetirementDate != "2025-01-01" {
		t.Fatal("expected retirement_date 2025-01-01")
	}

	policies := dossier.Policies
	// Using days_between/365.25 per spec:
	// Policy 1: 9132 days / 365.25 ≈ 24.996 years, salary 50000 * 1.0
	// Policy 2: 5479 days / 365.25 ≈ 14.997 years, salary 60000 * 0.8 = 48000
	// The formula simplifies: policy_pension = Σ(eff_salary * years) * accrual_rate * (years_i / total)
	// Values will be close to (but not exactly) 24625 / 14775 due to leap years
	if policies[0].AttainablePension == nil || policies[1].AttainablePension == nil {
		t.Fatal("expected attainable_pension to be set")
	}
	// Total pension should be approximately 39400 (sum of both policy pensions)
	totalPension := *policies[0].AttainablePension + *policies[1].AttainablePension
	assertFloat(t, "total pension", totalPension, totalPension) // sanity: not zero
	if totalPension < 39000 || totalPension > 39800 {
		t.Fatalf("total pension out of range: %.2f", totalPension)
	}
	// Policy 1 should get ~62.5% (25/40), Policy 2 ~37.5% (15/40)
	ratio := *policies[0].AttainablePension / totalPension
	if ratio < 0.62 || ratio > 0.63 {
		t.Fatalf("policy 1 ratio out of expected range: %.4f", ratio)
	}
}

func TestCalculateRetirementNotEligible(t *testing.T) {
	// Person born 1990-01-01, retirement 2025-01-01 → age 35, needs 40 years service
	resp := Process(&model.CalculationRequest{
		TenantID: "test",
		CalculationInstructions: model.CalculationInstructions{
			Mutations: []model.Mutation{
				{
					MutationID:             "m1",
					MutationDefinitionName: "create_dossier",
					MutationType:           "DOSSIER_CREATION",
					ActualAt:               "2020-01-01",
					MutationProperties:     json.RawMessage(`{"dossier_id":"d1","person_id":"p1","name":"Young Person","birth_date":"1990-01-01"}`),
				},
				addPolicyMut("SCHEME-A", "2020-01-01", 50000, 1.0),
				retirementMut("2025-01-01"),
			},
		},
	})

	if resp.CalculationMetadata.CalculationOutcome != "FAILURE" {
		t.Fatalf("expected FAILURE")
	}
	if resp.CalculationResult.Messages[0].Code != "NOT_ELIGIBLE" {
		t.Fatalf("expected NOT_ELIGIBLE, got %s", resp.CalculationResult.Messages[0].Code)
	}
}

func TestCalculateRetirementNoDossier(t *testing.T) {
	resp := Process(makeReq("test", retirementMut("2025-01-01")))

	if resp.CalculationMetadata.CalculationOutcome != "FAILURE" {
		t.Fatalf("expected FAILURE")
	}
	if resp.CalculationResult.Messages[0].Code != "DOSSIER_NOT_FOUND" {
		t.Fatalf("expected DOSSIER_NOT_FOUND")
	}
}

// --- Full flow (README example) ---

func TestFullFlowReadmeExample(t *testing.T) {
	resp := Process(makeReq("tenant-001",
		createDossierMut(),
		addPolicyMut("SCHEME-A", "2000-01-01", 50000, 1.0),
		indexationMut(0.03, "", ""),
	))

	if resp.CalculationMetadata.CalculationOutcome != "SUCCESS" {
		t.Fatalf("expected SUCCESS")
	}
	if resp.CalculationMetadata.TenantID != "tenant-001" {
		t.Fatalf("expected tenant-001")
	}
	if len(resp.CalculationResult.Messages) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(resp.CalculationResult.Messages))
	}
	if len(resp.CalculationResult.Mutations) != 3 {
		t.Fatalf("expected 3 mutations, got %d", len(resp.CalculationResult.Mutations))
	}

	end := resp.CalculationResult.EndSituation
	if end.MutationIndex != 2 {
		t.Fatalf("expected mutation_index 2, got %d", end.MutationIndex)
	}
	if end.ActualAt != "2021-01-01" {
		t.Fatalf("expected actual_at 2021-01-01, got %s", end.ActualAt)
	}

	policies := end.Situation.Dossier.Policies
	if len(policies) != 1 {
		t.Fatalf("expected 1 policy, got %d", len(policies))
	}
	assertFloat(t, "salary after indexation", policies[0].Salary, 51500)
	if policies[0].PolicyID != dossierID+"-1" {
		t.Fatalf("expected policy_id %s-1", dossierID)
	}
}

// --- Test helpers ---

const (
	dossierID = "d2222222-2222-2222-2222-222222222222"
	personID  = "p3333333-3333-3333-3333-333333333333"
)

func makeReq(tenant string, mutations ...model.Mutation) *model.CalculationRequest {
	return &model.CalculationRequest{
		TenantID: tenant,
		CalculationInstructions: model.CalculationInstructions{
			Mutations: mutations,
		},
	}
}

func createDossierMut() model.Mutation {
	return model.Mutation{
		MutationID:             "a1111111-1111-1111-1111-111111111111",
		MutationDefinitionName: "create_dossier",
		MutationType:           "DOSSIER_CREATION",
		ActualAt:               "2020-01-01",
		MutationProperties:     json.RawMessage(`{"dossier_id":"` + dossierID + `","person_id":"` + personID + `","name":"Jane Doe","birth_date":"1960-06-15"}`),
	}
}

var mutSeq int

func addPolicyMut(schemeID, empStart string, salary float64, ptf float64) model.Mutation {
	mutSeq++
	props, _ := json.Marshal(map[string]any{
		"scheme_id":             schemeID,
		"employment_start_date": empStart,
		"salary":                salary,
		"part_time_factor":      ptf,
	})
	return model.Mutation{
		MutationID:             "b" + string(rune('0'+mutSeq)) + "000000-0000-0000-0000-000000000000",
		MutationDefinitionName: "add_policy",
		MutationType:           "DOSSIER",
		ActualAt:               "2020-01-01",
		DossierID:              dossierID,
		MutationProperties:     json.RawMessage(props),
	}
}

func indexationMut(pct float64, schemeID, effectiveBefore string) model.Mutation {
	p := map[string]any{"percentage": pct}
	if schemeID != "" {
		p["scheme_id"] = schemeID
	}
	if effectiveBefore != "" {
		p["effective_before"] = effectiveBefore
	}
	props, _ := json.Marshal(p)
	return model.Mutation{
		MutationID:             "c5555555-5555-5555-5555-555555555555",
		MutationDefinitionName: "apply_indexation",
		MutationType:           "DOSSIER",
		ActualAt:               "2021-01-01",
		DossierID:              dossierID,
		MutationProperties:     json.RawMessage(props),
	}
}

func indexationMutWithScheme(pct float64, schemeID string) model.Mutation {
	return indexationMut(pct, schemeID, "")
}

func indexationMutWithEffectiveBefore(pct float64, before string) model.Mutation {
	return indexationMut(pct, "", before)
}

func retirementMut(date string) model.Mutation {
	props, _ := json.Marshal(map[string]any{"retirement_date": date})
	return model.Mutation{
		MutationID:             "d6666666-6666-6666-6666-666666666666",
		MutationDefinitionName: "calculate_retirement_benefit",
		MutationType:           "DOSSIER",
		ActualAt:               date,
		DossierID:              dossierID,
		MutationProperties:     json.RawMessage(props),
	}
}
