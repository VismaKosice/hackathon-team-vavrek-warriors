package mutations

import "pension-engine/internal/model"

// MutationHandler defines the contract for all mutation implementations.
// Execute validates and applies in a single call, returning patches directly.
type MutationHandler interface {
	Execute(state *model.Situation, mutation *model.Mutation) (msgs []model.CalculationMessage, hasCritical bool, fwdPatch, bwdPatch []byte)
}
