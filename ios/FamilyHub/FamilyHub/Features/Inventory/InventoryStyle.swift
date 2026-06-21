import SwiftUI

/// Maps the design's fixed icon/tint string sets onto SF Symbols and SwiftUI
/// colors, and lists the choices offered in the Add-Area / Add-Item sheets.
enum InventoryStyle {
    /// Icon keys offered in the Add-Area picker (matches the design handoff).
    static let icons: [String] = ["box", "drop", "cart", "pills", "sparkles", "heart", "cloud", "flame"]

    /// Tint keys offered in the Add-Area picker.
    static let tints: [String] = ["blue", "orange", "teal", "green", "purple", "indigo", "red"]

    /// Common units offered as chips in the Add-Item sheet.
    static let units: [String] = ["pcs", "pods", "bottles", "packs", "tins", "rolls", "boxes", "bags", "tubes", "sprays", "tablets", "sheets"]

    /// Orange is the universal "low stock" accent.
    static let low = Color(.systemOrange)

    static func symbol(_ icon: String) -> String {
        switch icon {
        case "drop":     return "drop.fill"
        case "cart":     return "cart.fill"
        case "pills":    return "pills.fill"
        case "sparkles": return "sparkles"
        case "heart":    return "heart.fill"
        case "cloud":    return "cloud.fill"
        case "flame":    return "flame.fill"
        case "box":      return "shippingbox.fill"
        default:         return "shippingbox.fill"
        }
    }

    static func color(_ tint: String) -> Color {
        switch tint {
        case "orange": return Color(.systemOrange)
        case "teal":   return Color(.systemTeal)
        case "green":  return Color(.systemGreen)
        case "purple": return Color(.systemPurple)
        case "indigo": return Color(.systemIndigo)
        case "red":    return Color(.systemRed)
        case "blue":   return Color(.systemBlue)
        default:       return Color(.systemBlue)
        }
    }
}
