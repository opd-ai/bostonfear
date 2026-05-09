# Camera Usability Verification

## Scope
Validates Workstream 4 camera interactions and fallback mode.

## Controls Under Test
- Keyboard: `[` (orbit CCW), `]` (orbit CW), `V` (toggle top-down fallback)
- Mouse: wheel up/down (orbit), middle click (toggle fallback)
- Touch: left-third tap (orbit CCW), right-third tap (orbit CW), center tap (toggle fallback)

## Verification Steps
1. Start server and desktop client.
2. Rotate through all 8 directions and confirm HUD reports `dir=1/8` through `dir=8/8`.
3. Trigger top-down fallback and verify board returns to unprojected top-down layout.
4. Perform move and action inputs in at least 3 camera directions and in top-down mode.
5. Confirm action routing remains unchanged (keyboard/touch actions still succeed).

## Hit-Test Accuracy Check
Touch move targets should continue to trigger the intended location actions regardless of camera mode because touch hit boxes remain bound to the canonical logical board rectangles.

## Error-Rate Comparison Procedure
1. Run one 5-minute session in top-down mode.
2. Run one 5-minute session in pseudo-3D mode.
3. Compare invalid action retry counts from HUD (`Invalid retries`) for relative parity.
