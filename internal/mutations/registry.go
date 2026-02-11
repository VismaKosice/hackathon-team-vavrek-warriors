package mutations

var registry = map[string]MutationHandler{
	"create_dossier": &CreateDossierHandler{},
}

func Get(name string) (MutationHandler, bool) {
	h, ok := registry[name]
	return h, ok
}
