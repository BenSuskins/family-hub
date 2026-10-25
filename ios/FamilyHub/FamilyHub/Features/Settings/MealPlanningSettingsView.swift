import SwiftUI
import FamilyHubKit

struct MealPlanningSettingsView: View {
    let apiClient: any APIClientProtocol

    @State private var weekday: MealWeekday = .friday
    @State private var savedWeekday: MealWeekday = .friday
    @State private var isLoading = true
    @State private var hasLoaded = false
    @State private var isSaving = false
    @State private var actionError: APIError?
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        Form {
            Section {
                Picker("Meal week starts on", selection: $weekday) {
                    ForEach(MealWeekday.allCases, id: \.self) { day in
                        Text(day.title).tag(day)
                    }
                }
                .accessibilityIdentifier("Meal week starts on")
                .disabled(!hasLoaded || isSaving)
            } header: { Text("Meal planning") } footer: {
                Text("Applies to everyone in your family. Saved meals keep their dates.")
            }.listRowBackground(AppTheme.surface)
        }
        .appListStyle()
        .navigationTitle("Meal planning")
        .navigationBarTitleDisplayMode(.large)
        .toolbar {
            ToolbarItem(placement: .confirmationAction) {
                Button(isSaving ? "Saving…" : "Save") { Task { await save() } }
                    .buttonStyle(AppCommitButtonStyle())
                    .accessibilityIdentifier("Save")
                    .disabled(weekday == savedWeekday || isSaving || !hasLoaded)
            }
        }
        .overlay {
            if isLoading {
                ProgressView()
            } else if !hasLoaded {
                ContentUnavailableView {
                    Label("Meal Planning Unavailable", systemImage: "calendar")
                } description: {
                    Text("Meal planning settings could not be loaded.")
                } actions: {
                    Button("Retry") { Task { await load() } }
                }
            }
        }
        .errorAlert($actionError)
        .task { await load() }
    }

    private func load() async {
        isLoading = true
        actionError = nil
        defer { isLoading = false }
        do {
            let settings = try await apiClient.fetchSettings()
            weekday = settings.mealWeekStart
            savedWeekday = settings.mealWeekStart
            hasLoaded = true
        } catch {
            actionError = .from(error)
        }
    }

    private func save() async {
        isSaving = true
        defer { isSaving = false }
        do {
            try await apiClient.updateMealWeekStart(weekday)
            dismiss()
        } catch {
            actionError = .from(error)
        }
    }

}
