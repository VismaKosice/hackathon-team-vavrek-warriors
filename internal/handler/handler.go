package handler

import (
	"encoding/json"
	"net/http"

	"pension-engine/internal/engine"
	"pension-engine/internal/model"
)

func HandleCalculation(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusBadRequest, "Method not allowed")
		return
	}

	var req model.CalculationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if len(req.CalculationInstructions.Mutations) == 0 {
		writeError(w, http.StatusBadRequest, "At least one mutation is required")
		return
	}

	resp := engine.Process(&req)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(model.ErrorResponse{
		Status:  status,
		Message: message,
	})
}
