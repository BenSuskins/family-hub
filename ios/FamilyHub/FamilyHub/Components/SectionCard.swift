import SwiftUI

/// Surface-coloured card with a header row (icon + title) and slot for content rows.
///
/// Usage:
///   SectionCard(icon: "checkmark.circle", iconColor: .teal, title: "Chores Due") {
///       ForEach(rows) { row in ChoreRow(chore: row) }
///   }
struct SectionCard<Content: View>: View {
    let icon: String
    let iconColor: Color
    let title: String
    let content: Content

    init(icon: String, iconColor: Color, title: String, @ViewBuilder content: () -> Content) {
        self.icon = icon
        self.iconColor = iconColor
        self.title = title
        self.content = content()
    }

    var body: some View {
        VStack(spacing: 0) {
            HStack(spacing: 8) {
                Image(systemName: icon)
                    .foregroundStyle(iconColor)
                    .font(.system(size: 15))
                Text(title)
                    .font(.system(size: 14, weight: .semibold))
                    .foregroundStyle(Theme.textPrimary)
                Spacer()
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 12)

            Rectangle()
                .fill(Theme.borderDivider)
                .frame(height: 1)

            content
        }
        .background(Theme.surface)
        .clipShape(RoundedRectangle(cornerRadius: 14))
    }
}
