# Workstream 6 Verification

## Scope
Verifies onboarding, turn/action clarity, reconnection visibility, and outcome feedback.

## Automated Checks
- `go test -race ./...`
- `go vet ./...`

## Manual Checks
1. Start server and client.
2. Confirm first-session onboarding appears and can be advanced with `ENTER` or skipped with `H`.
3. Disconnect server temporarily and verify state banner changes to reconnect/sync status.
4. Reconnect server and verify banner returns to synchronized state.
5. Execute actions and confirm the results panel shows:
   - action outcome summary
   - resource deltas
   - doom delta

## Forced Disconnect/Reconnect Drill
1. Start server and client.
2. Stop server process for 10-15 seconds.
3. Restart server.
4. Verify banner transitions:
   - `Reconnecting...` / `Syncing...`
   - back to synchronized state once connected.

## Turn and Outcome Comprehension Session
Ask a first-time participant to complete one turn and explain:
- whose turn it is
- actions remaining
- what changed after each action

Record notes under `docs/ux-observations/`.
