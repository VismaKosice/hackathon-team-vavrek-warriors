package handler

import (
	json "github.com/goccy/go-json"
	"github.com/valyala/fasthttp"

	"pension-engine/internal/engine"
	"pension-engine/internal/model"
)

func HandleCalculation(ctx *fasthttp.RequestCtx) {
	if !ctx.IsPost() {
		writeError(ctx, 400, "Method not allowed")
		return
	}

	if string(ctx.Path()) != "/calculation-requests" {
		ctx.SetStatusCode(404)
		return
	}

	var req model.CalculationRequest
	if err := json.Unmarshal(ctx.PostBody(), &req); err != nil {
		writeError(ctx, 400, "Invalid request body: "+err.Error())
		return
	}

	if len(req.CalculationInstructions.Mutations) == 0 {
		writeError(ctx, 400, "At least one mutation is required")
		return
	}

	resp := engine.Process(&req)

	ctx.SetContentType("application/json")
	body, _ := json.Marshal(resp)
	ctx.SetBody(body)
}

func writeError(ctx *fasthttp.RequestCtx, status int, message string) {
	ctx.SetContentType("application/json")
	ctx.SetStatusCode(status)
	body, _ := json.Marshal(model.ErrorResponse{
		Status:  status,
		Message: message,
	})
	ctx.SetBody(body)
}
