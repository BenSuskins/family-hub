import SwiftUI

/// Rounded-square tinted tile with a white glyph — the area's visual identity.
/// Used at several sizes (18 inline, 38 list, 52 header, 72 add-area preview).
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
            .fill(InventoryStyle.color(tint))
            .frame(width: size, height: size)
            .overlay {
                Image(systemName: InventoryStyle.symbol(icon))
                    .font(.system(size: size * 0.5, weight: .semibold))
                    .foregroundStyle(.white)
            }
            .overlay(
                RoundedRectangle(cornerRadius: size * 0.28, style: .continuous)
                    .stroke(Color.white.opacity(0.4), lineWidth: 0.5)
                    .blendMode(.overlay)
            )
    }
}
