import SwiftUI
import FamilyHubKit

extension Color {
    /// Builds a colour from one of the server's `RRGGBB` calendar colours.
    ///
    /// The app has the same initialiser in `AppTheme`. Only this two-line shim
    /// is duplicated — the parsing itself lives in ``HexColor`` in
    /// FamilyHubKit, which is the one module both the app and this extension
    /// can import. FamilyHubKit is deliberately Foundation-only, so the
    /// `Color` it can't express is built on each side.
    init?(hex: String) {
        guard let parsed = HexColor(hex: hex) else { return nil }
        self.init(red: parsed.red, green: parsed.green, blue: parsed.blue)
    }
}

/// Where a widget row sends you when it's tapped.
///
/// Non-optional on purpose: these are compile-time constants, and a widget that
/// silently drops its links is harder to notice than a build that won't produce
/// one. The app routes all three in `ContentView.onOpenURL`.
enum WidgetLink {
    static let today = URL(string: "familyhub://today")!
    static let calendar = URL(string: "familyhub://calendar")!
    static let meals = URL(string: "familyhub://meals")!
}
