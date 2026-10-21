import SwiftUI
import UIKit
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

enum WidgetPalette {
    private static func adaptive(light: UInt32, dark: UInt32) -> Color {
        Color(uiColor: UIColor { traits in
            let value = traits.userInterfaceStyle == .dark ? dark : light
            return UIColor(
                red: CGFloat((value >> 16) & 255) / 255,
                green: CGFloat((value >> 8) & 255) / 255,
                blue: CGFloat(value & 255) / 255,
                alpha: 1
            )
        })
    }

    static let background = adaptive(light: 0xf7f1e7, dark: 0x201f1a)
    static let surface = adaptive(light: 0xfffcf6, dark: 0x2d3028)
    static let ink = adaptive(light: 0x343c31, dark: 0xf7f0e2)
    static let secondary = adaptive(light: 0x646357, dark: 0xc3beaf)
    static let accent = adaptive(light: 0x3c654c, dark: 0xbad0ab)
    static let onAccent = adaptive(light: 0xffffff, dark: 0x201f1a)
    static let border = adaptive(light: 0xe6dfd2, dark: 0x494a3e)
    static let sage = adaptive(light: 0xe6eddd, dark: 0x3e4a37)
    static let clay = adaptive(light: 0x8b503c, dark: 0xe0baa0)
    static let departure = adaptive(light: 0xf0e4d2, dark: 0x3c3428)
}
