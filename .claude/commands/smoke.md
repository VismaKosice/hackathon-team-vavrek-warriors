Build Docker image and run a smoke test.

Steps:
1. Build: `docker build -t pension-engine /Users/lukasvavrek/Developer/hackathon-team-vavrek-warriors`
2. Stop any existing container: `docker rm -f pension-engine-test 2>/dev/null`
3. Run: `docker run -d --name pension-engine-test -p 8080:8080 pension-engine`
4. Wait for startup: `sleep 1`
5. Send test request:
```
curl -s -X POST http://localhost:8080/calculation-requests \
  -H "Content-Type: application/json" \
  -d '{
    "tenant_id": "test",
    "calculation_instructions": {
      "mutations": [{
        "mutation_id": "a1111111-1111-1111-1111-111111111111",
        "mutation_definition_name": "create_dossier",
        "mutation_type": "DOSSIER_CREATION",
        "actual_at": "2020-01-01",
        "mutation_properties": {
          "dossier_id": "d2222222-2222-2222-2222-222222222222",
          "person_id": "p3333333-3333-3333-3333-333333333333",
          "name": "Jane Doe",
          "birth_date": "1960-06-15"
        }
      }]
    }
  }'
```
6. Verify the response has HTTP 200, correct structure, and `calculation_outcome: "SUCCESS"`
7. Cleanup: `docker rm -f pension-engine-test`

Report the full response and whether it matches expectations.
