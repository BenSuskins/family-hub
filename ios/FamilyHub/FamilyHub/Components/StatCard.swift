import SwiftUI

/// Small stat card: label (top), large number, subtitle (bottom).
struct StatCard: View {
    let label: String
    let value: Int
    let subtitle: String
    var subtitleColor: Color = Theme.textMuted

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(label)
                .font(.system(size: 9, weight: .semibold))
                .textCase(.uppercase)
                .tracking(0.6)
                .foregroundStyle(Theme.textMuted)
            Text("\(value)")
                .font(.system(size: 26, weight: .bold))
                .foregroundStyle(Theme.textPrimary)
            Text(subtitle)
                .font(.system(size: 11))
                .foregroundStyle(subtitleColor)
        }
        .padding(12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(Theme.surface)
        .clipShape(RoundedRectangle(cornerRadius: 12))
    }
}

#Preview {
    HStack(spacing: 8) {
        StatCard(label: "Chores", value: 12, subtitle: "5 overdue", subtitleColor: Theme.statusRed)
        StatCard(label: "Events", value: 4, subtitle: "Next 7 days")
        StatCard(label: "Meals", value: 12, subtitle: "of 21 planned")
    }
    .padding()
    .background(Theme.background)
}
