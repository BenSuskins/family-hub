import SwiftUI

/// Keep stock controls independent of the row's edit action.
struct QuantityStepper: View {
    let quantity: Int
    var editableQuantity: Binding<Int>?
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize
    var isLow: Bool = false
    var label = "Quantity"
    let onDecrement: () -> Void
    let onIncrement: () -> Void

    var body: some View {
        HStack(spacing: 10) {
            stepButton(systemName: "minus", tint: Color(.secondaryLabel), disabled: quantity <= 0, action: onDecrement)

            if let editableQuantity {
                TextField("Quantity", value: editableQuantity, format: .number)
                    .keyboardType(.numberPad)
                    .multilineTextAlignment(.center)
                    .font(.title3.weight(.semibold)).monospacedDigit()
                    .foregroundStyle(isLow ? InventoryStyle.low : AppTheme.ink)
                    .frame(minWidth: 44, maxWidth: dynamicTypeSize.isAccessibilitySize ? .infinity : 90)
                    .accessibilityIdentifier("Count")
            } else {
                Text("\(quantity)")
                    .font(.title3.weight(.semibold)).monospacedDigit()
                    .foregroundStyle(isLow ? InventoryStyle.low : AppTheme.ink)
                    .frame(minWidth: 28)
                    .contentTransition(.numericText())
            }

            stepButton(systemName: "plus", tint: Color.accentColor, disabled: false, action: onIncrement)
        }
        .background(AppTheme.iconBackground, in: Capsule())
    }

    private func stepButton(systemName: String, tint: Color, disabled: Bool, action: @escaping () -> Void) -> some View {
        Button(action: action) {
            Image(systemName: systemName)
                .font(.body.weight(.semibold))
                .foregroundStyle(tint)
                .frame(width: 44, height: 44)

                .opacity(disabled ? 0.4 : 1)
                .contentShape(Circle())
        }
        .buttonStyle(.borderless)
        .disabled(disabled)
        .accessibilityLabel("\(systemName == "minus" ? "Decrease" : "Increase") \(label)")
        .sensoryFeedback(.selection, trigger: quantity)
    }
}
