import SwiftUI
import FamilyHubKit

struct ChoreRowView: View {
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize
    let chore: Chore
    let user: User?
    let apiClient: any APIClientProtocol
    let onComplete: () -> Void
    var onSkip: (() -> Void)?
    var minHeight: CGFloat = 56

    var body: some View {
        HStack(spacing: 12) {
            if !dynamicTypeSize.isAccessibilitySize {
                UserAvatar(user: user, size: 32, apiClient: apiClient).accessibilityHidden(true)
            }

            VStack(alignment: .leading, spacing: 2) {
                Text(chore.name)
                    .font(.body.weight(.medium))
                    .strikethrough(chore.status == .completed)
                    .foregroundStyle(chore.status == .completed ? .tertiary : .primary)
                    .fixedSize(horizontal: false, vertical: true)
                Text(user?.name ?? "Unassigned")
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                if let date = chore.formattedDueDate {
                    Text("Due \(date)")
                        .font(.caption)
                        .foregroundStyle(chore.status == .overdue ? .red : .secondary)
                } else if chore.status == .overdue {
                    Text("Overdue")
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(.red)
                }
            }

            Spacer(minLength: 0)

            if let onSkip {
                Menu {
                    Button("Skip this task", systemImage: "forward.end") { onSkip() }
                } label: {
                    Image(systemName: "ellipsis").frame(width: 44, height: 44)
                }
                .accessibilityLabel("Actions for \(chore.name)")
            }

            Button(action: onComplete) {
                CheckCircleView(isSelected: chore.status == .completed)
                    .frame(width: 44, height: 44)
            }
            .buttonStyle(.borderless)
            .accessibilityLabel(chore.status == .completed ? "Completed \(chore.name)" : "Complete \(chore.name)")
            .disabled(chore.status == .completed)
            .sensoryFeedback(.success, trigger: chore.status == .completed)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .frame(minHeight: minHeight)
    }
}
