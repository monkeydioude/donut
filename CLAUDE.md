# Donut

## Workflow

For a **new feature** (skip steps 1 to 3 for a fix):

1. Run `/grilling` on the feature before writing any code.
2. Write an ADR in `docs/adr/` with the decisions taken.
3. Write the test cases from that ADR and present them to the developer before coding.
4. Code it.
5. Run `/security-review` once coding is done.
6. Make sure all tests pass:
   - front: `cd client && npm run typecheck`
   - back: `cd backend && go test -race -v ./...`

For a **fix**: still run `/security-review` (step 5) and make all tests pass (step 6).
