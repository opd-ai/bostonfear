# Mobile Deployment Guide

This guide provides step-by-step instructions for building and deploying the BostonFear mobile client to Android and iOS devices.

## Prerequisites

### Common Requirements
- Go 1.24 or later
- Git (to clone the repository)
- BostonFear repository cloned locally

### Android Requirements
- Android Studio (latest stable version recommended)
- Android SDK with API level 29+ (Android 10 or later)
- Android NDK version 25.2.9519653 or compatible
- Java Development Kit (JDK) 17

### iOS Requirements
- macOS with Xcode 15 or later
- iOS 16+ deployment target
- Valid Apple Developer account (for physical device deployment)
- CocoaPods (optional, for dependency management)

## Installation

### Step 1: Install Go Mobile Tools

```bash
# Install gomobile and ebitenmobile
go install golang.org/x/mobile/cmd/gomobile@latest
go install github.com/hajimehoshi/ebiten/v2/cmd/ebitenmobile@latest

# Initialize gomobile
gomobile init

# Initialize ebitenmobile
ebitenmobile init
```

Verify installation:
```bash
ebitenmobile version
```

## Android Deployment

### Step 1: Build the AAR Library

From the BostonFear repository root:

```bash
# Create output directory
mkdir -p dist

# Build Android AAR
ebitenmobile bind -target android -o dist/bostonfear.aar ./cmd/mobile
```

This creates `dist/bostonfear.aar` (Android Archive library) that can be imported into Android Studio.

### Step 2: Create Android Studio Project

1. Open Android Studio
2. Create a new Android project or use an existing one
3. Choose "Empty Activity" template
4. Set minimum SDK to API 29 (Android 10)

### Step 3: Import AAR into Android Studio

1. Copy `dist/bostonfear.aar` to `app/libs/` in your Android project
2. Edit `app/build.gradle` and add:

```gradle
dependencies {
    implementation files('libs/bostonfear.aar')
    implementation 'androidx.appcompat:appcompat:1.6.1'
    // Add other required dependencies
}
```

3. Sync Gradle files

### Step 4: Configure MainActivity

Create or edit `MainActivity.java` (or `.kt` for Kotlin):

```java
package com.example.bostonfear;

import android.app.Activity;
import android.os.Bundle;
import go.Seq;
import mobile.Mobile;

public class MainActivity extends Activity {
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        
        // Initialize the Go mobile binding
        Seq.setContext(getApplicationContext());
        
        // Start the Ebitengine game
        // Note: The binding provides a runnable game loop
        Mobile.run(this);
    }
}
```

### Step 5: Configure Server URL

#### For Android Emulator
Use the special IP address that routes to the host machine:
```
ws://10.0.2.2:8080/ws
```

#### For Physical Android Device
Use your development machine's LAN IP address:
```
ws://192.168.1.100:8080/ws
```

Find your LAN IP:
- Linux/Mac: `ifconfig | grep inet`
- Windows: `ipconfig`

Ensure your phone and development machine are on the same WiFi network.

### Step 6: Build and Run

1. Connect an Android device via USB with USB debugging enabled, or start an emulator
2. Click "Run" in Android Studio (green play button)
3. Select your target device
4. The app will install and launch

### Troubleshooting Android

**Issue**: `ebitenmobile bind` fails with NDK errors
- **Solution**: Ensure ANDROID_NDK_HOME environment variable is set:
  ```bash
  export ANDROID_NDK_HOME=$ANDROID_SDK_ROOT/ndk/25.2.9519653
  ```

**Issue**: WebSocket connection fails from emulator
- **Solution**: Verify you're using `ws://10.0.2.2:8080/ws` not `localhost`
- Ensure the server is running on your host machine
- Check that no firewall is blocking port 8080

**Issue**: App crashes on startup
- **Solution**: Check Android Studio Logcat for Go runtime errors
- Verify minimum SDK is API 29+
- Ensure all required permissions are in `AndroidManifest.xml`

## iOS Deployment

### Step 1: Build the XCFramework

From the BostonFear repository root:

```bash
# Create output directory
mkdir -p dist

# Build iOS XCFramework
ebitenmobile bind -target ios -o dist/BostonFear.xcframework ./cmd/mobile
```

This creates `dist/BostonFear.xcframework` that can be imported into Xcode.

### Step 2: Create or Open Xcode Project

1. Open Xcode
2. Create a new iOS App project or use an existing one
3. Set deployment target to iOS 16.0 or later
4. Choose Swift or Objective-C as your language

### Step 3: Import XCFramework into Xcode

1. In Xcode, select your project in the navigator
2. Select your app target
3. Go to "General" tab → "Frameworks, Libraries, and Embedded Content"
4. Click "+" and select "Add Files..."
5. Navigate to `dist/BostonFear.xcframework` and add it
6. Ensure "Embed & Sign" is selected

### Step 4: Configure AppDelegate

For Swift:
```swift
import UIKit
import BostonFear

@main
class AppDelegate: UIResponder, UIApplicationDelegate {
    var window: UIWindow?

    func application(_ application: UIApplication, 
                    didFinishLaunchingWithOptions launchOptions: [UIApplication.LaunchOptionsKey: Any]?) -> Bool {
        // Start the Ebitengine game
        MobileRun()
        return true
    }
}
```

For Objective-C:
```objc
#import "AppDelegate.h"
#import <BostonFear/Mobile.h>

@implementation AppDelegate

- (BOOL)application:(UIApplication *)application didFinishLaunchingWithOptions:(NSDictionary *)launchOptions {
    MobileRun();
    return YES;
}

@end
```

### Step 5: Configure Server URL

iOS clients must use the host machine's local network IP address. You **cannot** use `localhost` or `127.0.0.1`.

Find your Mac's IP address:
```bash
ifconfig | grep "inet " | grep -v 127.0.0.1
```

Example server URL:
```
ws://192.168.1.100:8080/ws
```

Ensure your iOS device/simulator and development machine are on the same network.

### Step 6: Build and Run

#### For iOS Simulator:
1. Select a simulator device (iPhone 15, iPad Pro, etc.)
2. Click "Run" (play button) in Xcode
3. The simulator will launch and run the app

#### For Physical Device:
1. Connect your iPhone/iPad via USB
2. Trust the computer on your device when prompted
3. In Xcode, select your physical device from the device menu
4. You may need to configure signing:
   - Select your project → Target → "Signing & Capabilities"
   - Select your Apple Developer team
   - Xcode will provision a development certificate
5. Click "Run"
6. On first run, you may need to trust the developer certificate on your device:
   - Go to Settings → General → VPN & Device Management
   - Trust your developer certificate

### Troubleshooting iOS

**Issue**: `ebitenmobile bind` fails with Xcode errors
- **Solution**: Ensure Xcode command line tools are installed:
  ```bash
  xcode-select --install
  ```

**Issue**: WebSocket connection fails
- **Solution**: Verify you're using the correct LAN IP address, not `localhost`
- Ensure both device and Mac are on the same WiFi network
- Check that macOS firewall allows incoming connections on port 8080
- Try temporarily disabling macOS firewall to test

**Issue**: Build fails with signing errors
- **Solution**: You need a valid Apple Developer account
- Create a development provisioning profile in Xcode
- For testing only, you can use a free Apple Developer account (7-day certificate validity)

**Issue**: "Untrusted Developer" message on device
- **Solution**: Go to Settings → General → VPN & Device Management → Trust certificate

## Network Configuration Reference

### Server URL Matrix

| Environment | Server URL | Notes |
|-------------|------------|-------|
| Android Emulator | `ws://10.0.2.2:8080/ws` | Special emulator IP routing to host |
| Android Physical | `ws://192.168.1.X:8080/ws` | Replace X with your host machine's LAN IP |
| iOS Simulator | `ws://192.168.1.X:8080/ws` | Replace X with your Mac's LAN IP |
| iOS Physical | `ws://192.168.1.X:8080/ws` | Replace X with your Mac's LAN IP |

### Common Networking Issues

**CORS / Origin Policy**
- The Go WebSocket server accepts connections from any origin by default (development mode)
- For production, configure `allowedOrigins` in the server

**WebSocket Upgrade Failures**
- Ensure the server is running and accessible: `curl http://192.168.1.X:8080/health`
- Check server logs for connection attempts
- Verify no proxy or firewall is blocking WebSocket upgrades

**Timeout / Connection Refused**
- Ping your server from the device to verify network connectivity
- Ensure port 8080 is not blocked by firewall
- Try using HTTP (not HTTPS) for local development testing

**SSL/TLS Certificate Issues**
- For local development, use `ws://` (unencrypted WebSocket)
- For production, use `wss://` with proper SSL certificates
- Self-signed certificates require explicit trust configuration on mobile devices

## Performance Tips

### Build Optimization
- Use `-ldflags="-w -s"` to reduce binary size (strips debug info)
- Enable Go module caching to speed up rebuilds

### Runtime Optimization
- Profile the Go client with `pprof` to identify bottlenecks
- Monitor memory usage in Xcode Instruments (iOS) or Android Profiler
- Reduce asset sizes (sprites, audio) for faster loading

## Deployment Checklist

Before deploying to production:

- [ ] Replace placeholder sprites with final game assets
- [ ] Configure production WebSocket server URL
- [ ] Enable SSL/TLS for secure WebSocket connections (`wss://`)
- [ ] Set server `allowedOrigins` to restrict WebSocket upgrades
- [ ] Test on multiple device types and OS versions
- [ ] Verify touch targets are at least 44×44 logical pixels
- [ ] Test network reconnection and session token persistence
- [ ] Profile memory usage and frame rate under load
- [ ] Implement analytics for crash reporting and user metrics
- [ ] Prepare App Store / Google Play Store listings
- [ ] Review Apple App Store and Google Play Store submission guidelines

## Additional Resources

- [Ebitengine Mobile Documentation](https://ebitengine.org/en/documents/mobile.html)
- [gomobile Documentation](https://pkg.go.dev/golang.org/x/mobile/cmd/gomobile)
- [Android NDK Guide](https://developer.android.com/ndk/guides)
- [iOS App Distribution Guide](https://developer.apple.com/documentation/xcode/distributing-your-app-for-beta-testing-and-releases)

## iOS XCTest UI Automation

The CI validates iOS XCFramework structure and simulator boot, but full touch input verification requires an Xcode project with XCTest UI tests. Here's how to set this up for comprehensive testing:

### Step 1: Create Xcode Project

1. Open Xcode and create a new iOS App project:
   - Product Name: `BostonFearTest`
   - Organization Identifier: `com.example`
   - Interface: `Storyboard`
   - Language: `Swift`

2. Add XCFramework to the project:
   - Select your project in the navigator
   - Select the app target
   - Go to "General" → "Frameworks, Libraries, and Embedded Content"
   - Click "+" and add `BostonFear.xcframework`
   - Ensure "Embed & Sign" is selected

### Step 2: Create UI Test Target

1. File → New → Target → iOS UI Testing Bundle
2. Name it `BostonFearUITests`

### Step 3: Write XCTest UI Tests

Create `BostonFearUITests/TouchInputTests.swift`:

```swift
import XCTest

class TouchInputTests: XCTestCase {
    var app: XCUIApplication!
    
    override func setUpWithError() throws {
        continueAfterFailure = false
        app = XCUIApplication()
        app.launch()
        
        // Wait for game to load and connect
        sleep(3)
    }
    
    func testGatherAction() throws {
        // Calculate gather button coordinates (adjust based on your UI)
        let gatherButton = app.coordinate(withNormalizedOffset: CGVector(dx: 0.25, dy: 0.85))
        gatherButton.tap()
        
        // Wait for action to process
        sleep(1)
        
        // Verify action was accepted (check for resource change or state update)
        // This requires adding accessibility identifiers to your UI elements
    }
    
    func testInvestigateAction() throws {
        let investigateButton = app.coordinate(withNormalizedOffset: CGVector(dx: 0.50, dy: 0.85))
        investigateButton.tap()
        sleep(1)
    }
    
    func testTwoActionTurnLimit() throws {
        // Perform two actions
        let gatherButton = app.coordinate(withNormalizedOffset: CGVector(dx: 0.25, dy: 0.85))
        gatherButton.tap()
        sleep(1)
        
        gatherButton.tap()
        sleep(2)
        
        // Verify turn advances to next player
        // Check turn indicator or player name changes
    }
    
    func testReconnection() throws {
        // This test would require simulating network interruption
        // You can use Network Link Conditioner or XCTest network mocking
    }
}
```

### Step 4: Run UI Tests

```bash
# List available simulators
xcrun simctl list devices

# Run tests on specific simulator
xcodebuild test \
  -project BostonFearTest.xcodeproj \
  -scheme BostonFearTest \
  -destination 'platform=iOS Simulator,name=iPhone 15,OS=latest' \
  -only-testing:BostonFearUITests/TouchInputTests
```

### Step 5: Integrate with CI

Add to `.github/workflows/mobile.yml`:

```yaml
- name: Run XCTest UI tests
  run: |
    xcodebuild test \
      -project path/to/BostonFearTest.xcodeproj \
      -scheme BostonFearTest \
      -destination 'platform=iOS Simulator,name=iPhone 15' \
      -resultBundlePath TestResults
```

### Notes

- XCTest UI tests require coordinates or accessibility identifiers
- The ebitengine mobile binding may not expose UIKit accessibility hooks directly
- Alternative: Use `simctl` with coordinate-based tap commands:
  ```bash
  xcrun simctl io booted screenshot screenshot.png
  xcrun simctl io booted tap 200 600  # Tap at x=200, y=600
  ```
- Full XCTest integration is recommended for regression testing in production apps

See `scripts/ios-simulator-test.sh` for current CI validation approach.

## Support

For issues specific to BostonFear mobile deployment:
- Check `docs/MOBILE_VERIFICATION_RUNBOOK.md` for testing procedures
- Review server logs at `/tmp/bostonfear-server.log` (if configured)
- Enable verbose logging in the Go client for debugging

Estimated deployment time: **15-30 minutes** for developers familiar with mobile development tools.
