# Mobile Verification Runbook

**Last Updated**: 2026-05-19

## Scope
This runbook verifies the alpha mobile binding path for BostonFear and records platform constraints. These manual tests complement the automated CI validation performed in `.github/workflows/mobile.yml`.

### CI Validation Coverage (Automated)

The following mobile tests run automatically in CI on every push and pull request:

#### Android (✅ Automated in CI)
- **Library Binding**: AAR artifact builds successfully (`ebitenmobile bind -target android`)
- **Emulator Smoke Test**: Android emulator (API 29) boots and runs test APK (`.github/workflows/mobile.yml:45-236`)
- **Connection Verification**: Client connects to server and receives `connectionStatus` message via logcat monitoring
- **Touch Input**: Automated coordinate calculation and touch injection for Gather and Investigate actions via `scripts/android-touch-coords.sh`
- **Action Processing**: Logcat confirms action messages received and processed by client

#### iOS (✅ Automated in CI)
- **Library Binding**: XCFramework artifact builds successfully (`ebitenmobile bind -target ios`)
- **Framework Validation**: `scripts/ios-simulator-test.sh` validates XCFramework structure and binary linkability (`.github/workflows/mobile.yml:260-321`)
- **Simulator Boot**: iOS simulator boots successfully with latest runtime
- **Go-Level Tests**: Mobile binding tests pass with `GOOS=ios` on simulator

### Manual Testing Scope

The tests below complement CI automation and should be performed on physical devices or when validating device-specific issues not covered by emulator/simulator tests:

- **Physical device performance** (battery, thermal, network conditions)
- **Device-specific input quirks** (notch areas, safe area insets, gesture conflicts)
- **App store submission validation** (provisioning profiles, entitlements)
- **Full gameplay sessions** (15+ minutes continuous play)
- **Background/foreground transitions** and app lifecycle edge cases
- **Multiple network types** (WiFi, LTE, airplane mode toggle)

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
