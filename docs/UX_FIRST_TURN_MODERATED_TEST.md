# First-Turn Moderated Usability Test

## Objective
Validate that a first-time participant can complete one full turn without external documentation while correctly identifying:
- current turn owner
- actions remaining
- available actions
- action outcome feedback (including invalid action recovery)

## Scope
- Product surface: Ebitengine desktop client connected to the local Go server
- Session target: one full participant turn (exactly two actions)
- Success threshold: participant completes turn with no facilitator intervention beyond scripted prompts

## Moderator Script
1. Start the server: `go run . server --port 8082`
2. Start a client: `go run ./cmd/desktop -server ws://localhost:8082/ws`
3. Give only this prompt to participant:
   - "Please complete your turn using two actions and explain what happened after each action."
4. Do not explain game rules unless the participant is blocked for over 60 seconds.
5. Capture the observations in the worksheet below.

## Observation Worksheet
- Participant ID:
- Date:
- Build/commit:
- Could identify current turn owner within 30 seconds? (yes/no)
- Could identify actions remaining before first action? (yes/no)
- Could identify at least one available action without help? (yes/no)
- Completed first action without facilitator help? (yes/no)
- Understood first action result feedback? (yes/no)
- Completed second action and ended turn correctly? (yes/no)
- Invalid action attempted? (yes/no)
- If invalid action attempted, did participant recover without help? (yes/no/na)
- Notes:

## Pass/Fail Rule
Pass if all conditions are true:
1. Participant completes exactly two actions in one turn.
2. Participant identifies current turn and actions remaining without facilitator intervention.
3. Participant correctly explains at least one action outcome.

## Evidence Storage
Store completed worksheets in `docs/ux-observations/` as `YYYY-MM-DD-participant-<id>.md`.

## Current Status
- Test procedure defined and ready for moderated execution.
- Pending external participant sessions to collect first-time-user evidence.
