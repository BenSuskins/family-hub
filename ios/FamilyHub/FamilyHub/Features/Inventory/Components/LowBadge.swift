import SwiftUI

/// Small uppercase "LOW" pill shown beside an item that's at or below its par.
struct LowBadge: View {
    var body: some View {
        Text("Low")
            .font(.system(size: 11, weight: .bold))
            .textCase(.uppercase)
            .kerning(0.3)
            .foregroundStyle(InventoryStyle.low)
            .padding(.horizontal, 6)
            .padding(.vertical, 1)
            .background(InventoryStyle.low.opacity(0.14), in: RoundedRectangle(cornerRadius: 5, style: .continuous))
    }
}
