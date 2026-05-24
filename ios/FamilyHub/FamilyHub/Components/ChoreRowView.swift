import SwiftUI

struct ChoreRowView: View {
    let chore: Chore
    let user: User?
    let apiClient: any APIClientProtocol
    let onComplete: () -> Void
    var minHeight: CGFloat = 56
    var showDateWhenOverdue: Bool = false

    var body: some View {
        HStack(spacing: 12) {
            UserAvatar(user: user, size: 32, apiClient: apiClient)

            VStack(alignment: .leading, spacing: 2) {
                Text(chore.name)
                    .font(.system(size: 16, weight: .medium))
                    .strikethrough(chore.status == .completed)
                    .foregroundStyle(chore.status == .completed ? .tertiary : .primary)
                    .lineLimit(1)
                HStack(spacing: 4) {
                    if let user {
                        Text(user.name)
                            .font(.system(size: 13))
                            .foregroundStyle(.secondary)
                        Text("·")
                            .font(.system(size: 13))
                            .foregroundStyle(.secondary)
                    }
                    if chore.status == .overdue {
                        Text("Overdue")
                            .font(.system(size: 13, weight: .semibold))
                            .foregroundStyle(.red)
                        if showDateWhenOverdue, let date = chore.formattedDueDate {
                            Text(date)
                                .font(.system(size: 13))
                                .foregroundStyle(.secondary)
                        }
                    } else if let date = chore.formattedDueDate {
                        Text(date)
                            .font(.system(size: 13))
                            .foregroundStyle(.secondary)
                    }
                }
            }

            Spacer(minLength: 0)

            Button(action: onComplete) {
                CheckCircleView(isSelected: chore.status == .completed)
            }
            .buttonStyle(.plain)
            .sensoryFeedback(.success, trigger: chore.status == .completed)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .frame(minHeight: minHeight)
    }
}
