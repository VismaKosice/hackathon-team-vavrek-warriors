package mutations

var registry = map[string]MutationHandler{
	"create_dossier":               &CreateDossierHandler{},
	"add_policy":                   &AddPolicyHandler{},
	"apply_indexation":             &ApplyIndexationHandler{},
	"calculate_retirement_benefit": &CalculateRetirementBenefitHandler{},
	"project_future_benefits":      &ProjectFutureBenefitsHandler{},
}

func Get(name string) (MutationHandler, bool) {
	h, ok := registry[name]
	return h, ok
}
