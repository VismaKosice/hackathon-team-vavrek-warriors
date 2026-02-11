package mutations

import "pension-engine/internal/model"

// MutationHandler defines the contract for all mutation implementations.
// Each mutation validates business rules and applies state changes.
type MutationHandler interface {
	Validate(state *model.Situation, mutation *model.Mutation) []model.CalculationMessage
	Apply(state *model.Situation, mutation *model.Mutation) []model.CalculationMessage
}
