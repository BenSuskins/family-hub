import SwiftUI

/// A compact horizontal fill bar for level-tracked items (e.g. how full a bottle
/// is). The fill turns orange when the item is low, otherwise green; the trailing
/// label shows the percentage.
struct LevelBar: View {
    let level: Int
    var isLow: Bool = false
    var barWidth: CGFloat = 64

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
            .frame(width: barWidth, height: 8)

            Text("\(level)%")
                .font(.subheadline.weight(.semibold))
                .monospacedDigit()
                .foregroundStyle(tint)
                .frame(minWidth: 38, alignment: .trailing)
        }
    }
}

struct LevelStepper: View {
    let level: Int
    let isLow: Bool
    let label: String
    let onDecrement: () -> Void
    let onIncrement: () -> Void

    var body: some View {
        HStack(spacing: 4) {
            stepButton(systemName: "minus", disabled: level <= 0, action: onDecrement)
            LevelBar(level: level, isLow: isLow, barWidth: 36)
            stepButton(systemName: "plus", disabled: level >= 100, action: onIncrement)
        }
    }

    private func stepButton(systemName: String, disabled: Bool, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Image(systemName: systemName)
                .font(.body.weight(.semibold))
                .foregroundStyle(systemName == "minus" ? AppTheme.secondary : AppTheme.accent)
                .frame(width: 44, height: 44)
                .contentShape(Circle())
        }
        .buttonStyle(.borderless)
        .disabled(disabled)
        .opacity(disabled ? 0.4 : 1)
        .accessibilityLabel("\(systemName == "minus" ? "Decrease" : "Increase") \(label) level by 5 percentage points")
        .sensoryFeedback(.selection, trigger: level)
    }
}
