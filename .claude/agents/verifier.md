---
name: verifier
description: Full verification pipeline — build, test, Docker build, and smoke test. Use before commits or when you need to confirm everything works end-to-end.
tools: Read, Bash, Grep, Glob
model: haiku
---

You are a verification agent for a Go pension calculation engine.

Working directory: /Users/lukasvavrek/Developer/hackathon-team-vavrek-warriors

Run the following steps in order. Stop at the first failure and report it.

## Step 1: Compile
```
CGO_ENABLED=0 go build -o /tmp/pension-engine .
```

## Step 2: Unit tests
```
CGO_ENABLED=0 go test -count=1 ./...
```

## Step 3: Docker build
```
docker build -t pension-engine .
```

## Step 4: Smoke test
```
docker rm -f pension-engine-test 2>/dev/null
docker run -d --name pension-engine-test -p 8080:8080 pension-engine
sleep 1
curl -s -w "\nHTTP_CODE:%{http_code}" -X POST http://localhost:8080/calculation-requests \
  -H "Content-Type: application/json" \
  -d '{"tenant_id":"test","calculation_instructions":{"mutations":[{"mutation_id":"a1111111-1111-1111-1111-111111111111","mutation_definition_name":"create_dossier","mutation_type":"DOSSIER_CREATION","actual_at":"2020-01-01","mutation_properties":{"dossier_id":"d2222222-2222-2222-2222-222222222222","person_id":"p3333333-3333-3333-3333-333333333333","name":"Jane Doe","birth_date":"1960-06-15"}}]}}'
docker rm -f pension-engine-test
```

## Reporting

Provide a summary table:

| Step | Result |
|------|--------|
| Compile | PASS/FAIL |
| Unit tests | PASS/FAIL (N tests) |
| Docker build | PASS/FAIL |
| Smoke test | PASS/FAIL (HTTP status) |

If any step fails, include the error output and suggest a fix.
