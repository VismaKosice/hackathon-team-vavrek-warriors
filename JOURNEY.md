# From Zero to 54,000 req/s: Winning the Visma Performance Hackathon

## The Challenge

**Visma Performance Hackathon** - a one-day competition to build a high-performance pension calculation engine. Single HTTP endpoint, five mutations, correctness first, then speed.

**Scoring:** 40 pts correctness + 40 pts performance + 30 pts bonus + 5 pts code quality = **115 total**

**Our result: 114.5/115 points. First place.**

| Category | Score | Max |
|----------|-------|-----|
| Correctness | 40 | 40 |
| Performance | 40 | 40 |
| Bonus Features | 30 | 30 |
| Code Quality | 4.5 | 5 |
| **Total** | **114.5** | **115** |

**Final throughput: 53,964 req/s** | **Cold start: 134ms** | **~1,100 lines of Go**

---

## The Timeline

Here's the full journey, commit by commit, over a single day of building.

---

### Phase 1: Foundation (11:00 - 11:10)

#### Commit `15803c7` - Go Skeleton + Create Dossier
> *The first 10 minutes*

Started with a clean Go project: `net/http` server, `encoding/json`, `google/uuid`. Implemented `create_dossier` mutation with validation (duplicate dossier, empty name, invalid/future birth date).

**Architecture decisions made early:**
- Handler interface pattern (`MutationHandler`) with a registry map - no switch/case
- Clean separation: `handler/` (HTTP) -> `engine/` (orchestration) -> `mutations/` (business logic)
- Domain model: `Situation` -> `Dossier` -> `Persons[]` + `Policies[]`

```
main.go                    # Entry point
internal/
  handler/handler.go       # HTTP layer
  engine/engine.go         # Mutation processing loop
  model/                   # Request, response, domain types
  mutations/
    mutation.go            # MutationHandler interface
    registry.go            # Name -> handler map
    create_dossier.go      # First handler
```

#### Commit `001e6eb` - Core Mutations Complete
> *+10 minutes, all correctness tests passing*

Implemented the remaining three core mutations in a single commit:
- **add_policy**: Sequential policy ID generation (`{dossier_id}-{seq}`), salary/part-time validation
- **apply_indexation**: Scheme and date filtering with AND logic, negative salary clamping
- **calculate_retirement_benefit**: Days/365.25 formula, eligibility (age >= 65 OR service >= 40), proportional distribution

14 tests covering all mutations, error cases, and the full happy path.

**Score at this point: 80/115** (40 correctness + 40 performance estimated baseline)

---

### Phase 2: Bonus Features (11:10 - 14:25)

#### Commit `d424b73` - Future Benefits Projection (Bonus)
> *+5 pts bonus*

Implemented `project_future_benefits` - projecting pension at regular intervals from start to end date. Same formula as retirement calculation, minus the eligibility check.

**Score: 85/115** (+5 bonus)

#### Commit `11b671e` - JSON Patch (RFC 6902)
> *+16 pts bonus, but a performance trap*

Each processed mutation now includes forward and backward JSON patches describing state transitions. Built a generic diff algorithm comparing situation snapshots.

This worked correctly but introduced a severe performance regression - we didn't know it yet.

**Score: ~101/115** (+16 bonus for JSON patch)

#### Commit `863dd2a` - Scheme Registry Integration
> *+5 pts bonus, performance partially recovered*

External HTTP service providing per-scheme accrual rates. Implementation highlights:
- `sync.Map` for thread-safe caching (fetch once, use forever)
- Concurrent goroutine fetching for multiple scheme IDs
- 2-second timeout with 0.02 fallback rate
- CA certificates in Docker image for HTTPS

Also optimized the JSON patch: replaced `json.Marshal`/`json.Unmarshal` roundtrip with direct struct-to-map conversion.

**Score: 101/115** (40 + 40 + 21 bonus) | **Throughput: 20,321 req/s**

---

### Phase 3: The Performance Journey (13:45 - 15:35)

This is where it gets interesting. We had all the features, but our throughput had dropped from an estimated ~50k req/s to 20k because of the generic JSON patch diffing.

#### The Problem: Generic Diffing is Expensive

The JSON patch implementation worked by:
1. Converting `Situation` struct to `map[string]interface{}` (via marshal + unmarshal)
2. Running recursive tree diff between before/after states
3. Doing this **for every mutation** in every request

Each conversion allocated ~30-50 heap objects. For a 5-mutation request, that's 300+ allocations just for diffing - more than the actual business logic.

#### Commit `d5671c3` - The Big Performance Commit
> *Foundation: fasthttp, goccy/go-json, merged Execute*

Seven optimizations in one commit:

| Optimization | Impact | What Changed |
|---|---|---|
| `net/http` -> `valyala/fasthttp` | +30-40% | Buffer reuse via sync.Pool, no goroutine-per-request |
| `encoding/json` -> `goccy/go-json` | +20-30% | 2-3x faster marshal/unmarshal, drop-in replacement |
| Merged Validate + Apply into Execute | +10-15% | Parse mutation properties once instead of twice |
| `google/uuid` -> `math/rand` UUID | +1-3% | Eliminated crypto/rand syscall per request |
| Pre-allocated slices | +5-8% | `make([]T, 0, n)` based on mutation count |
| Custom `fastFormatDate` | +2-3% | Avoid `time.Format` layout parsing per projection |
| `fmt.Sprintf` -> string concat | +1-2% | Eliminated format parsing on hot paths |
| `-gcflags="-B"` in Dockerfile | +2-3% | Disabled bounds checking in compiled binary |

**Key insight:** The `MutationHandler` interface change was critical. Old:
```go
type MutationHandler interface {
    Validate(state, mutation) []CalculationMessage
    Apply(state, mutation) []CalculationMessage
}
```
New:
```go
type MutationHandler interface {
    Execute(state, mutation) (msgs, hasCritical, fwdPatch, bwdPatch)
}
```

One parse, one pass. Each handler owns its complete lifecycle.

#### Commit `7e3fa6a` - Mutation-Aware Patches
> *The single biggest performance win: eliminate all generic diffing*

**The insight:** Each mutation handler *knows exactly which fields it modifies*. `create_dossier` sets `/dossier`. `add_policy` adds to `/dossier/policies/-`. `apply_indexation` changes specific salary values. Why diff the entire state tree when we can emit targeted patches?

Before (generic diffing):
```go
// For EVERY mutation:
beforeMap := structToMap(state)     // 30+ allocations
handler.Execute(state, mutation)
afterMap := structToMap(state)      // 30+ more allocations
patches := DiffBoth(before, after)  // recursive tree walk
```

After (mutation-aware):
```go
// Inside create_dossier handler:
fwd := marshalPatches([]patchOp{
    {Op: "replace", Path: "/dossier", Value: marshalValue(state.Dossier)},
})
bwd := marshalPatches([]patchOp{
    {Op: "replace", Path: "/dossier", Value: jsonNull},
})
return nil, false, fwd, bwd
```

**Results:**
- Deleted `internal/jsonpatch/diff.go` entirely (198 lines)
- Removed `situationToGeneric` conversion from engine
- Net: **+110 lines added, -328 lines deleted**
- Each handler generates exactly the patches it needs - zero wasted work

#### Commit `a02df6a` - GC Tuning + Compiler Flags
> *Throughput: 20k -> 53k req/s*

Three runtime optimizations:

**1. Garbage Collector Tuning**
```go
debug.SetGCPercent(2000)                  // Default 100 -> 2000
debug.SetMemoryLimit(512 * 1024 * 1024)   // 512 MiB safety net
```
Default Go GC triggers when heap doubles (GOGC=100). At 2000, the heap can grow 20x before GC kicks in. Under sustained load, this means GC runs ~20x less frequently. `GOMEMLIMIT` prevents unbounded growth.

**2. Bounds Check Elimination**
```dockerfile
# Before: only main package
-gcflags="-B"

# After: ALL packages including dependencies
-gcflags="all=-B"
```
The `all=` prefix was critical - without it, only the main package skipped bounds checks. With it, fasthttp, go-json, and all internal packages benefit too.

**3. fasthttp Server Configuration**
```go
server := &fasthttp.Server{
    Handler:          handler.HandleCalculation,
    DisableKeepalive: false,
    ReadBufferSize:   8192,
    WriteBufferSize:  8192,
}
```

#### Commit `73b8141` - Fast Date Parser
> *Throughput: 53k -> 54k req/s*

Replaced all 7 `time.Parse("2006-01-02", s)` calls with a hand-rolled parser:

```go
func fastParseDate(s string) (time.Time, bool) {
    if len(s) != 10 || s[4] != '-' || s[7] != '-' {
        return time.Time{}, false
    }
    y := int(s[0]-'0')*1000 + int(s[1]-'0')*100 + int(s[2]-'0')*10 + int(s[3]-'0')
    m := time.Month(int(s[5]-'0')*10 + int(s[6]-'0'))
    d := int(s[8]-'0')*10 + int(s[9]-'0')
    if m < 1 || m > 12 || d < 1 || d > 31 {
        return time.Time{}, false
    }
    return time.Date(y, m, d, 0, 0, 0, 0, time.UTC), true
}
```

`time.Parse` does layout string parsing on every call - matching `2006` to year, `01` to month, etc. Our dates are always `YYYY-MM-DD`, so direct byte arithmetic is ~10x faster.

---

## Performance Progression

| Commit | What Changed | Throughput | Delta |
|--------|-------------|-----------|-------|
| `001e6eb` | Baseline (net/http + encoding/json) | ~14k req/s | - |
| `d5671c3` | fasthttp + go-json + merged Execute | ~50k req/s | +257% |
| `11b671e` | Added JSON Patch (generic diffing) | ~14k req/s | -72% |
| `863dd2a` | Optimized diffing + scheme registry | 20,321 req/s | +45% |
| `7e3fa6a` | Mutation-aware patches | ~20k req/s | ~0% |
| `a02df6a` | GC tuning + all=-B + server config | 53,403 req/s | +167% |
| `73b8141` | Fast date parser | 54,032 req/s | +1% |

The dramatic jump at `a02df6a` came from **GOGC=2000 combined with the earlier removal of generic diffing**. The mutation-aware patches had already eliminated the allocation pressure, but the GC was still running frequently from the default GOGC=100. Setting GOGC=2000 let the low-allocation-rate code actually benefit from having less garbage.

---

## Architecture: Final State

```
~1,100 lines of Go total

main.go (36 lines)
  └── GC tuning + fasthttp server

internal/handler/handler.go (48 lines)
  └── Unmarshal request, call engine, marshal response

internal/engine/engine.go (155 lines)
  └── Mutation processing loop, UUID generation

internal/mutations/ (462 lines total)
  ├── mutation.go        # MutationHandler interface
  ├── registry.go        # Name -> handler map
  ├── patch.go           # Shared patch helpers + fast date parser
  ├── create_dossier.go
  ├── add_policy.go
  ├── apply_indexation.go
  ├── calculate_retirement_benefit.go
  └── project_future_benefits.go

internal/schemeregistry/registry.go (108 lines)
  └── External scheme lookup with caching

internal/model/ (123 lines)
  └── Request, response, domain types

Dockerfile (12 lines)
  └── Multi-stage build, scratch base, static binary
```

---

## Key Takeaways

### 1. Profile Before You Optimize
The generic JSON patch diffing killed performance by 3.5x. We didn't profile first - we saw the throughput drop and had to reason backwards. Always measure.

### 2. Domain Knowledge Beats Generic Algorithms
A generic tree diff handles any two JSON objects. But each mutation handler *knows* it only touches 1-3 fields. Exploiting this domain knowledge eliminated 200+ lines of code AND made it faster.

### 3. GC Tuning is a Multiplier, Not a Fix
Setting GOGC=2000 on allocation-heavy code just delays the problem. Setting GOGC=2000 on allocation-light code (after mutation-aware patches) is a genuine 2.5x throughput boost.

### 4. Compiler Flags Matter - But Scope Matters More
`-gcflags="-B"` only affects the main package. `-gcflags="all=-B"` affects everything including your hot-path dependencies (fasthttp, go-json). The `all=` prefix was a meaningful win.

### 5. The 80/20 of Go Performance
The biggest wins came from just three changes:
- **fasthttp** over net/http (~2x)
- **goccy/go-json** over encoding/json (~1.5x)
- **GOGC=2000** with low-allocation code (~2.5x)

Everything else (fast UUID, date parser, string concat) was single-digit percentages.

### 6. Correctness First, Performance Second - But Architecture Enables Both
The mutation handler interface made it easy to swap between generic and mutation-aware patches without touching the engine or HTTP layer. Good abstractions make performance work safer.

---

## Tech Stack

| Component | Choice | Why |
|-----------|--------|-----|
| Language | Go | Static binary, minimal runtime, predictable perf |
| HTTP | valyala/fasthttp | Buffer pooling, 2x faster than net/http |
| JSON | goccy/go-json | 2-3x faster, drop-in replacement |
| UUID | math/rand | Avoids crypto/rand syscall |
| Container | Multi-stage, scratch base | 134ms cold start, minimal image |
| GC | GOGC=2000 + GOMEMLIMIT=512MiB | Minimal GC under sustained load |
| Build | `-gcflags="all=-B" -ldflags="-s -w"` | No bounds checks, stripped binary |

---

## Tools

Built with **Claude Code** (Claude Opus 4.6) as a pair programming partner throughout the entire hackathon day - from initial architecture through final optimization.

---

*Total development time: ~5 hours. From empty repo to 54,000 requests/second.*
