import SwiftUI
import FamilyHubKit

/// Preserve the selected area colour on a soft surface.
struct AreaIconTile: View {
    let icon: String
    let tint: String
    var size: CGFloat = 38

    init(icon: String, tint: String, size: CGFloat = 38) {
        self.icon = icon
        self.tint = tint
        self.size = size
    }

    init(area: InventoryArea, size: CGFloat = 38) {
        self.icon = area.icon
        self.tint = area.tint
        self.size = size
    }

    var body: some View {
        RoundedRectangle(cornerRadius: size * 0.28, style: .continuous)
            .fill(AppTheme.iconBackground)
            .frame(width: size, height: size)
            .overlay {
                Image(systemName: InventoryStyle.symbol(icon))
                    .font(.system(size: size * 0.5, weight: .semibold))
                    .foregroundStyle(InventoryStyle.color(tint))
            }
            .overlay(
                RoundedRectangle(cornerRadius: size * 0.28, style: .continuous)
                    .stroke(AppTheme.border, lineWidth: 0.5)
            )
    }
}
