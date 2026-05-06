# App Store Readiness Plan

## Issues

### 1. PrivacyInfo.xcprivacy — BLOCKING

Apple requires a privacy manifest for all App Store submissions since May 2024.
The app must declare any use of "required reason" APIs and data collection practices.

**APIs in use that require declaration:**
- `UserDefaults` (via app group `group.uk.co.suskins.familyhub`) — reason `CA92.1`: access defaults written by own app/extension
- `PhotosUI.PhotosPicker` — out-of-process on iOS 16+, no required-reason declaration needed

**Data collection:** none — this is a private family app with no analytics, no ads, no third-party SDKs.

**File to create:** `ios/FamilyHub/FamilyHub/PrivacyInfo.xcprivacy`

The manifest must be added as a resource in the Xcode target (not just dropped in the filesystem).

**Content:**
```xml
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>NSPrivacyAccessedAPITypes</key>
    <array>
        <dict>
            <key>NSPrivacyAccessedAPIType</key>
            <string>NSPrivacyAccessedAPICategoryUserDefaults</string>
            <key>NSPrivacyAccessedAPITypeReasons</key>
            <array>
                <string>CA92.1</string>
            </array>
        </dict>
    </array>
    <key>NSPrivacyCollectedDataTypes</key>
    <array/>
    <key>NSPrivacyTracking</key>
    <false/>
</dict>
</plist>
```

Steps:
1. Create the file at `ios/FamilyHub/FamilyHub/PrivacyInfo.xcprivacy`
2. In Xcode: File → Add Files → select `PrivacyInfo.xcprivacy`, ensure "Add to target: FamilyHub" is checked
   - Or add it to `project.pbxproj` manually under the FamilyHub target's resources build phase

---

### 2. Bundle ID inconsistency — non-blocking

| Location | Value |
|----------|-------|
| `PRODUCT_BUNDLE_IDENTIFIER` | `uk.co.suskins.FamilyHub` |
| App group | `group.uk.co.suskins.familyhub` |
| URL scheme | `com.familyhub.app` |

The URL scheme doesn't need to match the bundle ID, but `com.familyhub.app` is inconsistent with the `uk.co.suskins` namespace used everywhere else. This could cause confusion if another app registers the same scheme.

**Decision needed:** Keep `com.familyhub.app` or rename to `uk.co.suskins.familyhub`?

Steps (if renaming):
1. Update the URL scheme in `project.pbxproj` (search for `com.familyhub.app` under `CFBundleURLSchemes`)
2. Update any server-side or auth redirect URIs that reference the old scheme
3. Update `OIDCConfig` / callback URL handling in the iOS app

---

## Sequencing

| Step | Task | Blocking? |
|------|------|-----------|
| 1 | Create and wire `PrivacyInfo.xcprivacy` | Yes |
| 2 | Verify it appears under the target's resource build phase | Yes |
| 3 | Archive and validate in Xcode Organizer (checks manifest format) | Yes |
| 4 | Decide on URL scheme rename | No |
| 5 | Apply rename if decided | No |

## Out of scope

- App Store Connect metadata (screenshots, description, age rating) — content work, not engineering
- Push notifications — not implemented
- App Review notes — needed at submission time but no engineering changes required
