# Mobile Verification Runbook

## Scope
This runbook verifies the alpha mobile binding path for BostonFear and records platform constraints.

## Minimum Support Matrix

| Platform | Runtime Target | Minimum OS | Toolchain | Notes |
|---|---|---|---|---|
| Android | Emulator or physical device | Android 10 (API 29+) | Go 1.24+, gomobile, ebitenmobile, Android SDK | Use `ws://10.0.2.2:8080/ws` on emulator |
| iOS | Simulator or physical device | iOS 16+ | Go 1.24+, gomobile, ebitenmobile, Xcode 15+ | Use host machine LAN IP for server URL |

## Build Artifacts

1. Android binding:
   - `ebitenmobile bind -target android -o dist/bostonfear.aar ./cmd/mobile`
2. iOS binding:
   - `ebitenmobile bind -target ios -o dist/BostonFear.xcframework ./cmd/mobile`

## Touch Input Parity Checklist

Run on both Android and iOS:

- [ ] Tap each neighborhood region and confirm move action is sent to the expected location.
- [ ] Tap action bar regions for `gather`, `investigate`, `ward`, `focus`, `research`, and `closegate`.
- [ ] Confirm touch targets are comfortably selectable (>=44px logical target).
- [ ] Confirm two-action turn cap still applies with touch-only play.

## Reconnect Token Reclaim Checklist

Run on both Android and iOS:

- [ ] Connect once and capture `connectionStatus` with a non-empty token.
- [ ] Force disconnect (disable network or stop server for 10-15 seconds).
- [ ] Restore network/server and reconnect with the saved token.
- [ ] Confirm the same player slot is reclaimed (ID/resources/location unchanged).
- [ ] Confirm actions can be submitted after reclaim without app restart.

## Mobile Smoke Checklist

Run on both Android and iOS:

- [ ] Connect to server and receive initial `gameState`.
- [ ] Complete one full turn with exactly 2 actions.
- [ ] Verify doom/resource updates are rendered after each action.
- [ ] Disconnect and reconnect once; verify state sync and slot reclaim.
- [ ] Drive game to a win or lose condition and verify terminal state display.

## Evidence to Capture

- Device/emulator model and OS version.
- Toolchain versions (`go version`, `gomobile version`, `ebitenmobile version`, Xcode/SDK level).
- Build command outputs for `.aar` and `.xcframework`.
- Short screen recording or screenshots for turn flow and reconnect reclaim.
