import SwiftUI

/// The easy increment control: − / count / + with circular tappable buttons.
/// The minus button is disabled (and a no-op) at 0; the count turns orange when
/// the item is low. Uses `.borderless` buttons so taps never trigger an
/// enclosing row's navigation.
struct QuantityStepper: View {
    let quantity: Int
    var isLow: Bool = false
    let onDecrement: () -> Void
    let onIncrement: () -> Void

    var body: some View {
        HStack(spacing: 10) {
            stepButton(systemName: "minus", tint: Color(.secondaryLabel), disabled: quantity <= 0, action: onDecrement)

            Text("\(quantity)")
                .font(.system(size: 19, weight: .semibold))
                .monospacedDigit()
                .foregroundStyle(isLow ? InventoryStyle.low : Color(.label))
                .frame(minWidth: 28)
                .contentTransition(.numericText())

            stepButton(systemName: "plus", tint: Color.accentColor, disabled: false, action: onIncrement)
        }
    }

    private func stepButton(systemName: String, tint: Color, disabled: Bool, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Image(systemName: systemName)
                .font(.system(size: 16, weight: .semibold))
                .foregroundStyle(tint)
                .frame(width: 34, height: 34)
                .background(
                    Circle()
                        .fill(Color(.secondarySystemGroupedBackground))
                        .overlay(Circle().stroke(Color(.separator), lineWidth: 0.5))
                )
                .opacity(disabled ? 0.4 : 1)
                .contentShape(Circle())
        }
        .buttonStyle(.borderless)
        .disabled(disabled)
        .sensoryFeedback(.selection, trigger: quantity)
    }
}
