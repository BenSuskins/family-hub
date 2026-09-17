import Foundation

/// A colour parsed from the `RRGGBB` strings the server stores against iCal
/// subscriptions, as plain components.
///
/// It lives here rather than beside the app's `Color(hex:)` because the widget
/// extension needs the same parsing and cannot import the app target —
/// FamilyHubKit is the one module both reach. Keeping it Foundation-only also
/// means the Linux CI job tests the parsing, which it could never do for a
/// SwiftUI `Color` extension.
public struct HexColor: Equatable {
    public let red: Double
    public let green: Double
    public let blue: Double

    /// Parses `#RRGGBB` or `RRGGBB`, either case. Returns nil for anything
    /// else — a three-digit short form, a colour name, an empty string — so
    /// callers fall back to their own tint rather than drawing black.
    public init?(hex: String) {
        let cleaned = hex
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .replacingOccurrences(of: "#", with: "")
        guard cleaned.count == 6, let rgb = UInt64(cleaned, radix: 16) else { return nil }
        red = Double((rgb >> 16) & 0xFF) / 255
        green = Double((rgb >> 8) & 0xFF) / 255
        blue = Double(rgb & 0xFF) / 255
    }
}
