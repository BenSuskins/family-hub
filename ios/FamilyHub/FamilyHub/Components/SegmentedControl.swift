import SwiftUI

struct SegmentedControl<Option: Hashable>: View {
    let options: [Option]
    @Binding var selection: Option
    let label: (Option) -> String
    var badge: (Option) -> Int? = { _ in nil }
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize

    var body: some View {
        Picker("Filter", selection: $selection) {
            ForEach(options, id: \.self) { option in
                Text(title(option)).tag(option)
            }
        }
        .modifier(AdaptivePickerStyle(isAccessibilitySize: dynamicTypeSize.isAccessibilitySize))
    }

    private func title(_ option: Option) -> String {
        if let count = badge(option), count > 0 {
            return "\(label(option)) (\(count))"
        }
        return label(option)
    }
}

struct AdaptivePickerStyle: ViewModifier {
    let isAccessibilitySize: Bool

    @ViewBuilder
    func body(content: Content) -> some View {
        if isAccessibilitySize {
            content.pickerStyle(.menu)
        } else {
            content.pickerStyle(.segmented)
        }
    }
}
