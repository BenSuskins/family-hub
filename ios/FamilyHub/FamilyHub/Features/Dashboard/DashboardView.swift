import SwiftUI

struct DashboardView: View {
    @State private var viewModel: DashboardViewModel
    @State private var showProfile = false

    init(apiClient: any APIClientProtocol) {
        _viewModel = State(wrappedValue: DashboardViewModel(apiClient: apiClient))
    }

    var body: some View {
        NavigationStack {
            ZStack(alignment: .top) {
                Theme.background.ignoresSafeArea()
                ScrollView {
                    VStack(spacing: 0) {
                        if case .failed(let error) = viewModel.state {
                            inlineError(error.localizedDescription)
                        }
                        if case .loaded(let stats) = viewModel.state {
                            dashboardContent(stats)
                        } else if case .loading = viewModel.state {
                            ProgressView()
                                .tint(Theme.textSecondary)
                                .padding(.top, 60)
                        }
                    }
                    .padding(.horizontal, 14)
                }
                .refreshable { await viewModel.load() }
            }
            .navigationTitle("Overview")
            .navigationBarTitleDisplayMode(.large)
            .toolbarBackground(Theme.background, for: .navigationBar)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button {
                        showProfile = true
                    } label: {
                        // Intentional stub: shows "?" until OIDC claim storage is wired up (out of scope).
                        // This is expected — do not treat the "?" placeholder as a bug.
                        UserAvatar(user: nil, size: 32)
                    }
                    .buttonStyle(.plain)
                }
            }
            .sheet(isPresented: $showProfile) {
                ProfileView()
            }
        }
        .task { await viewModel.load() }
    }

    // MARK: - Content

    @ViewBuilder
    private func dashboardContent(_ stats: DashboardStats) -> some View {
        VStack(spacing: 14) {
            // Stat row
            HStack(spacing: 8) {
                StatCard(
                    label: "Chores",
                    value: stats.choresDueToday + stats.choresOverdue,
                    subtitle: stats.choresOverdue > 0 ? "\(stats.choresOverdue) overdue" : "None overdue",
                    subtitleColor: stats.choresOverdue > 0 ? Theme.statusRed : Theme.textMuted
                )
                StatCard(label: "Events", value: 0, subtitle: "Next 7 days")
                StatCard(label: "Meals", value: 0, subtitle: "of 21 planned")
            }
            .padding(.top, 8)

            // Chores Due section
            let allDue = stats.choresOverdueList + stats.choresDueTodayList
            if !allDue.isEmpty {
                SectionCard(icon: "checkmark.circle", iconColor: .teal, title: "Chores Due") {
                    ForEach(allDue) { chore in
                        choreDueRow(chore)
                    }
                }
            }

            // Today's Meals section
            SectionCard(icon: "flame", iconColor: Theme.statusAmber, title: "Today's Meals") {
                mealRow(label: "Lunch", name: nil)
                mealRow(label: "Dinner", name: nil)
            }

            // Leaderboard section (placeholder — requires server endpoint)
            SectionCard(icon: "trophy", iconColor: Theme.statusAmber, title: "Leaderboard") {
                Text("Leaderboard coming soon")
                    .font(.system(size: 13))
                    .foregroundStyle(Theme.textMuted)
                    .padding(.horizontal, 14)
                    .padding(.vertical, 12)
            }
        }
        .padding(.bottom, 20)
    }

    // MARK: - Row helpers

    private func choreDueRow(_ chore: Chore) -> some View {
        VStack(spacing: 0) {
            HStack(spacing: 10) {
                UserAvatar(user: viewModel.users[chore.assignedToUserID ?? ""], size: 32)
                VStack(alignment: .leading, spacing: 2) {
                    Text(chore.name)
                        .font(.system(size: 14, weight: .medium))
                        .foregroundStyle(Theme.textPrimary)
                    HStack(spacing: 6) {
                        if let user = viewModel.users[chore.assignedToUserID ?? ""] {
                            Text(user.name)
                                .font(.system(size: 12))
                                .foregroundStyle(Theme.textSecondary)
                        }
                        if let variant = chore.badgeVariant {
                            StatusBadge(variant: variant)
                        }
                        if let date = chore.formattedDueDate {
                            Text(date)
                                .font(.system(size: 11))
                                .foregroundStyle(chore.status == .overdue ? Theme.statusRed : Theme.statusAmber)
                        }
                    }
                }
                Spacer()
                DoneButton(choreID: chore.id, viewModel: viewModel)
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 10)

            Rectangle()
                .fill(Theme.borderDivider)
                .frame(height: 1)
                .padding(.leading, 14 + 32 + 10)
        }
    }

    private func mealRow(label: String, name: String?) -> some View {
        VStack(spacing: 0) {
            HStack(spacing: 12) {
                Text(label)
                    .font(.system(size: 12))
                    .foregroundStyle(Theme.textMuted)
                    .frame(width: 50, alignment: .leading)
                Text(name ?? "— not planned —")
                    .font(.system(size: 14))
                    .foregroundStyle(name != nil ? Theme.textPrimary : Theme.textMuted)
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 10)

            Rectangle()
                .fill(Theme.borderDivider)
                .frame(height: 1)
        }
    }

    private func inlineError(_ message: String) -> some View {
        Text(message)
            .font(.system(size: 13))
            .foregroundStyle(Theme.statusRed)
            .padding(.horizontal, 14)
            .padding(.vertical, 8)
    }
}

// MARK: - Done button

private struct DoneButton: View {
    let choreID: String
    let viewModel: DashboardViewModel
    @State private var isLoading = false
    @State private var errorMessage: String?

    var body: some View {
        VStack(spacing: 4) {
            Button {
                Task {
                    isLoading = true
                    do {
                        try await viewModel.apiClient.completeChore(id: choreID)
                        await viewModel.load()
                    } catch {
                        errorMessage = error.localizedDescription
                    }
                    isLoading = false
                }
            } label: {
                HStack(spacing: 4) {
                    if isLoading {
                        ProgressView().scaleEffect(0.7).tint(Theme.statusGreen)
                    } else {
                        Image(systemName: "checkmark")
                            .font(.system(size: 11, weight: .semibold))
                            .foregroundStyle(Theme.statusGreen)
                        Text("Done")
                            .font(.system(size: 11, weight: .medium))
                            .foregroundStyle(Theme.statusGreen)
                    }
                }
                .padding(.horizontal, 9)
                .padding(.vertical, 5)
                .background(Theme.doneButtonBg)
                .overlay(
                    RoundedRectangle(cornerRadius: 8)
                        .stroke(Theme.doneButtonBorder, lineWidth: 1)
                )
                .clipShape(RoundedRectangle(cornerRadius: 8))
            }
            .buttonStyle(.plain)
            .disabled(isLoading)

            if let errorMessage {
                Text(errorMessage)
                    .font(.system(size: 9))
                    .foregroundStyle(Theme.statusRed)
                    .lineLimit(2)
                    .frame(maxWidth: 80)
            }
        }
    }
}
