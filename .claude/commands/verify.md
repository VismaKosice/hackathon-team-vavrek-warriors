Full verification pipeline. Run all steps and report results.

1. **Compile:** `cd /Users/lukasvavrek/Developer/hackathon-team-vavrek-warriors && go build -o pension-engine .`
2. **Unit tests:** `go test -v -count=1 ./...`
3. **Docker build:** `docker build -t pension-engine /Users/lukasvavrek/Developer/hackathon-team-vavrek-warriors`
4. **Smoke test:** Run the smoke test from the smoke command
5. **Report:** Summary of all results with pass/fail for each step

If any step fails, stop and fix before continuing.
