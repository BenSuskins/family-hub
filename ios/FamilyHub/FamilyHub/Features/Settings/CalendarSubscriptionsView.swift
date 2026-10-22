import SwiftUI
import FamilyHubKit

@Observable
final class CalendarSubscriptionsViewModel {
    var subscriptions: [CalendarSubscription] = []
    var isLoading = false
    var actionError: APIError?
    var refreshingIDs: Set<String> = []

    private let apiClient: any APIClientProtocol

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
    }

    func load() async {
        isLoading = true
        defer { isLoading = false }
        do {
            subscriptions = try await apiClient.fetchCalendarSubscriptions()
        } catch {
            actionError = .from(error)
        }
    }

    func create(name: String, url: String, color: String) async {
        do {
            let created = try await apiClient.createCalendarSubscription(name: name, url: url, color: color)
            subscriptions.append(created)
        } catch {
            actionError = .from(error)
        }
    }

    func updateColor(id: String, color: String) async {
        do {
            let updated = try await apiClient.updateCalendarSubscriptionColor(id: id, color: color)
            replace(updated)
        } catch {
            actionError = .from(error)
            await load()
        }
    }

    func refresh(id: String) async {
        refreshingIDs.insert(id)
        defer { refreshingIDs.remove(id) }
        do {
            try await apiClient.refreshCalendarSubscription(id: id)
            await load()
        } catch {
            actionError = .from(error)
        }
    }

    func delete(id: String) async {
        do {
            try await apiClient.deleteCalendarSubscription(id: id)
            subscriptions.removeAll { $0.id == id }
        } catch {
            actionError = .from(error)
        }
    }

    private func replace(_ updated: CalendarSubscription) {
        guard let index = subscriptions.firstIndex(where: { $0.id == updated.id }) else { return }
        subscriptions[index] = updated
    }
}

struct CalendarSubscriptionsView: View {
    let apiClient: any APIClientProtocol

    @State private var viewModel: CalendarSubscriptionsViewModel
    @State private var showingCreate = false
    @State private var pendingDeletion: CalendarSubscription?

    init(apiClient: any APIClientProtocol) {
        self.apiClient = apiClient
        _viewModel = State(initialValue: CalendarSubscriptionsViewModel(apiClient: apiClient))
    }

    var body: some View {
        List {
            Section {
                Text("Add and manage the external calendars shown in Family Hub.")
                    .font(.subheadline)
                    .foregroundStyle(AppTheme.secondary)
            }.listRowBackground(Color.clear)

            if viewModel.subscriptions.isEmpty && !viewModel.isLoading {
                Section {
                    ContentUnavailableView("No calendars", systemImage: "calendar.badge.plus", description: Text("Tap + to add an iCal feed."))
                }.listRowBackground(AppTheme.surface)
            }

            Section("Calendar feeds") {
                ForEach(viewModel.subscriptions) { subscription in
                    VStack(alignment: .leading, spacing: 12) {
                        VStack(alignment: .leading, spacing: 4) {
                            Text(subscription.name).font(.body.weight(.semibold))
                            Text(subscription.url)
                                .font(.caption)
                                .foregroundStyle(AppTheme.secondary)
                                .lineLimit(2)
                                .textSelection(.enabled)
                            Text(subscription.lastFetchedAt.map { "Updated \($0.formatted(date: .abbreviated, time: .shortened))" } ?? "Not refreshed yet")
                                .font(.caption)
                                .foregroundStyle(AppTheme.secondary)
                        }

                        Picker("Color", selection: Binding(
                            get: { subscription.color },
                            set: { color in Task { await viewModel.updateColor(id: subscription.id, color: color) } }
                        )) {
                            ForEach(CalendarSubscriptionColor.allCases) { color in
                                Text(color.rawValue.capitalized).tag(color.rawValue)
                            }
                        }

                        HStack {
                            Button {
                                Task { await viewModel.refresh(id: subscription.id) }
                            } label: {
                                if viewModel.refreshingIDs.contains(subscription.id) {
                                    ProgressView().controlSize(.small)
                                } else {
                                    Label("Refresh", systemImage: "arrow.clockwise")
                                }
                            }
                            .disabled(viewModel.refreshingIDs.contains(subscription.id))
                            .accessibilityIdentifier("Refresh calendar \(subscription.id)")

                            Spacer()

                            Button(role: .destructive) {
                                pendingDeletion = subscription
                            } label: {
                                Label("Delete", systemImage: "trash")
                            }
                            .accessibilityIdentifier("Delete calendar \(subscription.id)")
                        }
                    }
                    .padding(.vertical, 6)
                    .listRowBackground(AppTheme.surface)
                }
            }
        }
        .appListStyle()
        .navigationTitle("Calendars")
        .navigationBarTitleDisplayMode(.large)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                Button { showingCreate = true } label: { Image(systemName: "plus") }
                    .accessibilityLabel("Add Calendar")
            }
        }
        .overlay {
            if viewModel.isLoading && viewModel.subscriptions.isEmpty { ProgressView() }
        }
        .refreshable { await viewModel.load() }
        .sheet(isPresented: $showingCreate) {
            CalendarSubscriptionForm { name, url, color in
                viewModel.actionError = nil
                await viewModel.create(name: name, url: url, color: color)
                let error = viewModel.actionError
                viewModel.actionError = nil
                return error
            }
        }
        .confirmationDialog("Delete \(pendingDeletion?.name ?? "calendar")?", isPresented: Binding(
            get: { pendingDeletion != nil },
            set: { if !$0 { pendingDeletion = nil } }
        ), titleVisibility: .visible) {
            if let subscription = pendingDeletion {
                Button("Delete Calendar", role: .destructive) { Task { await viewModel.delete(id: subscription.id) } }
            }
        }
        .errorAlert($viewModel.actionError)
        .task { await viewModel.load() }
    }
}

private struct CalendarSubscriptionForm: View {
    @Environment(\.dismiss) private var dismiss
    @State private var name = ""
    @State private var url = ""
    @State private var color = CalendarSubscriptionColor.indigo.rawValue
    @State private var error: APIError?

    let create: (String, String, String) async -> APIError?

    var body: some View {
        NavigationStack {
            Form {
                Section("Calendar feed") {
                    TextField("Name", text: $name)
                        .textInputAutocapitalization(.words)
                        .accessibilityIdentifier("Calendar name")
                    TextField("iCal URL", text: $url)
                        .keyboardType(.URL)
                        .textInputAutocapitalization(.never)
                        .autocorrectionDisabled()
                        .accessibilityIdentifier("Calendar URL")
                    Picker("Color", selection: $color) {
                        ForEach(CalendarSubscriptionColor.allCases) { option in
                            Text(option.rawValue.capitalized).tag(option.rawValue)
                        }
                    }
                }
                Section {
                    Text("Use an HTTP or HTTPS iCal feed URL.")
                        .font(.footnote)
                        .foregroundStyle(AppTheme.secondary)
                }
            }
            .navigationTitle("Add Calendar")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Cancel") { dismiss() } }
                ToolbarItem(placement: .confirmationAction) {
                    Button("Add") { Task { await save() } }
                        .disabled(name.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty || url.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
            }
            .errorAlert($error)
        }
    }

    private func save() async {
        let result = await create(name.trimmingCharacters(in: .whitespacesAndNewlines), url.trimmingCharacters(in: .whitespacesAndNewlines), color)
        if let result { error = result } else { dismiss() }
    }
}
