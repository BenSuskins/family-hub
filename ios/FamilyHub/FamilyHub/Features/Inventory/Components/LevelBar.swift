import SwiftUI

/// A compact horizontal fill bar for level-tracked items (e.g. how full a bottle
/// is). The fill turns orange when the item is low, otherwise green; the trailing
/// label shows the percentage. Display-only — editing happens via the item form.
struct LevelBar: View {
    let level: Int
    var isLow: Bool = false

    private var fraction: CGFloat { CGFloat(min(100, max(0, level))) / 100 }
    private var tint: Color { isLow ? InventoryStyle.low : Color.appGreen }

    var body: some View {
        HStack(spacing: 8) {
            GeometryReader { geo in
                ZStack(alignment: .leading) {
                    Capsule()
                        .fill(Color(.tertiarySystemFill))
                    Capsule()
                        .fill(tint)
                        .frame(width: geo.size.width * fraction)
                }
            }
            .frame(width: 64, height: 8)

            Text("\(level)%")
                .font(.system(size: 14, weight: .semibold))
                .monospacedDigit()
                .foregroundStyle(tint)
                .frame(minWidth: 38, alignment: .trailing)
        }
    }
}
