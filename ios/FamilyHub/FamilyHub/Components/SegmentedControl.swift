import SwiftUI
import UIKit

/// A pill-style segmented control matching the app's design tokens, with an
/// optional numeric badge per option (e.g. an overdue count). Selecting a
/// segment animates the highlight and updates the binding.
struct SegmentedControl<Option: Hashable>: View {
    let options: [Option]
    @Binding var selection: Option
    let label: (Option) -> String
    /// Optional badge count shown next to a segment's label. Return `nil` (the
    /// default) or a non-positive value to hide it.
    var badge: (Option) -> Int? = { _ in nil }

    var body: some View {
        HStack(spacing: 2) {
            ForEach(options, id: \.self) { option in
                Button {
                    withAnimation(.spring(duration: 0.2)) { selection = option }
                } label: {
                    segment(option, isSelected: selection == option)
                }
                .buttonStyle(.plain)
                .foregroundStyle(Color.primary)
            }
        }
        .padding(2)
        .background(Color(UIColor.quaternarySystemFill), in: RoundedRectangle(cornerRadius: 9))
    }

    private func segment(_ option: Option, isSelected: Bool) -> some View {
        HStack(spacing: 4) {
            Text(label(option))
                .font(.system(size: 13, weight: .medium))
            if let count = badge(option), count > 0 {
                Text("\(count)")
                    .font(.system(size: 11, weight: .bold))
                    .foregroundStyle(.white)
                    .padding(.horizontal, 5)
                    .padding(.vertical, 1)
                    .background(Color.red, in: Capsule())
            }
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 7)
        .background(
            isSelected ? Color(UIColor.secondarySystemGroupedBackground) : Color.clear,
            in: RoundedRectangle(cornerRadius: 7)
        )
        .shadow(color: isSelected ? .black.opacity(0.08) : .clear, radius: 1, x: 0, y: 1)
    }
}
