# Pension Calculation Engine - Go

## Project Overview
High-performance pension calculation engine for the Visma Performance Hackathon.
Single HTTP endpoint: `POST /calculation-requests` on port 8080.

## Tech Stack
- **Language:** Go
- **HTTP:** github.com/valyala/fasthttp
- **JSON:** github.com/goccy/go-json (drop-in, faster than encoding/json)
- **UUID:** math/rand (fast UUID v4, no external dep)
- **Docker:** Multi-stage build, scratch base, static binary

## Project Structure
```
main.go                          # Entry point, fasthttp server
internal/
  handler/handler.go             # HTTP request/response handling
  engine/engine.go               # Core mutation processing loop
  model/
    request.go                   # Request types
    response.go                  # Response types
    situation.go                 # Domain: Dossier, Person, Policy
    message.go                   # CalculationMessage
  mutations/
    mutation.go                  # MutationHandler interface (returns patches)
    registry.go                  # Name-based registry (no if/else)
    patch.go                     # patchOp helpers for mutation-aware patches
    create_dossier.go
    add_policy.go
    apply_indexation.go
    calculate_retirement_benefit.go
    project_future_benefits.go
  schemeregistry/
    registry.go                  # External scheme registry client (cached)
```

## Conventions
- All mutation handlers implement `MutationHandler` interface
- Mutations registered in registry map — no switch/case dispatching
- Dates are strings in `YYYY-MM-DD` format, parsed only when needed for calculation
- Monetary values are `float64` (test tolerance is 0.01)
- Policy ID format: `{dossier_id}-{sequence_number}`
- CRITICAL errors halt processing; WARNING errors are recorded and processing continues
- HTTP 200 for all processed calculations (SUCCESS and FAILURE)

## Performance Priorities
1. Fast JSON serialization/deserialization
2. Minimal allocations (pre-allocate slices, reuse buffers)
3. Efficient policy filtering for apply_indexation
4. Single-pass calculations where possible
5. Sub-100ms cold start (static binary, no init overhead)

## Commands
- `/project:build` — compile the binary
- `/project:test` — run all Go tests
- `/project:smoke` — Docker build + curl smoke test
- `/project:verify` — full verification pipeline

## Workflow
1. Make changes
2. Run tests (`go test ./...`)
3. Verify with smoke test
4. Commit after each meaningful increment
