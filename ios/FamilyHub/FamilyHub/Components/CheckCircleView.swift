import SwiftUI

/// Animated circle checkbox indicator. Wrap in a Button when tappable.
struct CheckCircleView: View {
    let isSelected: Bool
    var size: CGFloat = 28
    var color: Color = Color.appGreen
    var unselectedBorderColor: Color = Color.appGreen

    var body: some View {
        ZStack {
            Circle()
                .strokeBorder(isSelected ? color : unselectedBorderColor, lineWidth: 1.5)
                .frame(width: size, height: size)
                .opacity(isSelected ? 0 : 1)
            Circle()
                .fill(color)
                .frame(width: size, height: size)
                .opacity(isSelected ? 1 : 0)
            Image(systemName: "checkmark")
                .font(.system(size: size * 0.46, weight: .semibold))
                .foregroundStyle(.white)
                .opacity(isSelected ? 1 : 0)
        }
    }
}
