import SwiftUI

/// All design-system colour tokens for the dark-navy theme.
enum Theme {
    static let background      = Color(hex: "0f172a")
    static let surface         = Color(hex: "1e293b")
    static let surfaceElevated = Color(hex: "334155")
    static let borderDivider   = Color(hex: "0f172a") // matches background — creates slot-gap between surface rows
    static let textPrimary     = Color(hex: "f1f5f9")
    static let textSecondary   = Color(hex: "94a3b8")
    static let textMuted       = Color(hex: "475569")
    static let accent          = Color(hex: "60a5fa")
    static let statusRed       = Color(hex: "ef4444")
    static let statusAmber     = Color(hex: "f59e0b")
    static let statusGreen     = Color(hex: "4ade80")
    static let doneButtonBg    = Color(hex: "1e3a2f")
    static let doneButtonBorder = Color(hex: "16a34a").opacity(0.2)
    static let avatarFallback  = Color(hex: "6366f1")
}

extension Color {
    init(hex: String) {
        let hex = hex.trimmingCharacters(in: CharacterSet.alphanumerics.inverted)
        var value: UInt64 = 0
        Scanner(string: hex).scanHexInt64(&value)
        let red   = Double((value >> 16) & 0xFF) / 255
        let green = Double((value >> 8)  & 0xFF) / 255
        let blue  = Double(value         & 0xFF) / 255
        self.init(red: red, green: green, blue: blue)
    }
}
