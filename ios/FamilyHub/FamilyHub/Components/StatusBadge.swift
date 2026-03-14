import SwiftUI

struct StatusBadge: View {
    enum Variant: Equatable {
        case overdue
        case dueToday
        case dueSoon

        var label: String {
            switch self {
            case .overdue:  return "Overdue"
            case .dueToday: return "Today"
            case .dueSoon:  return "Due Soon"
            }
        }

        var textColor: Color {
            switch self {
            case .overdue:          return Theme.statusRed
            case .dueToday, .dueSoon: return Theme.statusAmber
            }
        }

        var backgroundColor: Color { textColor.opacity(0.13) }
    }

    let variant: Variant

    var body: some View {
        Text(variant.label)
            .font(.system(size: 10, weight: .semibold))
            .foregroundStyle(variant.textColor)
            .padding(.horizontal, 6)
            .padding(.vertical, 2)
            .background(variant.backgroundColor)
            .clipShape(RoundedRectangle(cornerRadius: 4))
    }
}

// MARK: - Chore convenience

extension Chore {
    var badgeVariant: StatusBadge.Variant? {
        switch status {
        case .overdue:   return .overdue
        case .completed: return nil
        case .pending:
            guard let dueDate else { return .dueSoon }
            let date = ISO8601DateFormatter().date(from: dueDate)
                ?? parseShortDate(dueDate)
            guard let date else { return .dueSoon }
            return Calendar.current.isDateInToday(date) ? .dueToday : .dueSoon
        }
    }

    private func parseShortDate(_ string: String) -> Date? {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        formatter.locale = Locale(identifier: "en_US_POSIX")
        return formatter.date(from: String(string.prefix(10)))
    }
}

#Preview {
    HStack(spacing: 8) {
        StatusBadge(variant: .overdue)
        StatusBadge(variant: .dueToday)
        StatusBadge(variant: .dueSoon)
    }
    .padding()
    .background(Theme.background)
}
