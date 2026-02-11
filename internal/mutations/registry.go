package mutations

var registry = map[string]MutationHandler{
	"create_dossier":               &CreateDossierHandler{},
	"add_policy":                   &AddPolicyHandler{},
	"apply_indexation":             &ApplyIndexationHandler{},
	"calculate_retirement_benefit": &CalculateRetirementBenefitHandler{},
}

func Get(name string) (MutationHandler, bool) {
	h, ok := registry[name]
	return h, ok
}
