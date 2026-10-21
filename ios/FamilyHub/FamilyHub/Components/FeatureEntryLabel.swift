import SwiftUI

struct FeatureEntryLabel: View {
    let title: String
    let subtitle: String
    let systemImage: String
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize

    var body: some View {
        HStack(spacing: 16) {
            if !dynamicTypeSize.isAccessibilitySize {
                WarmIcon(symbol: systemImage)
            }
            VStack(alignment: .leading, spacing: 4) {
                Text(title)
                    .font(.headline)
                    .foregroundStyle(AppTheme.ink)
                Text(subtitle)
                    .font(.subheadline)
                    .foregroundStyle(AppTheme.secondary)
            }
            .fixedSize(horizontal: false, vertical: true)
        }
        .frame(minHeight: 56)
        .padding(.vertical, 4)
        .accessibilityElement(children: .combine)
    }
}
