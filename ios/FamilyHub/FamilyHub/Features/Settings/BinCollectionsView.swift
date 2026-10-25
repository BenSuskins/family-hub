import SwiftUI
import UniformTypeIdentifiers
import FamilyHubKit

struct BinCollectionsView: View {
    let apiClient: any APIClientProtocol
    @State private var settings: BinScheduleSettings?
    @State private var isLoading = false
    @State private var isUploading = false
    @State private var showingImporter = false
    @State private var actionError: APIError?
    @State private var statusMessage: String?

    var body: some View {
        Form {
            Section {
                if let settings, settings.collectionCount > 0 {
                    LabeledContent("Calendar loaded", value: "\(settings.collectionCount) dates")
                }
                Text("Upload the ICS calendar from your council. Family Hub uses it for the Home card and 7pm reminders.")
                    .font(.subheadline)
                    .foregroundStyle(AppTheme.secondary)
                Link("Download the Braintree Route 17 calendar", destination: URL(string: "https://www.braintree.gov.uk/homepage/495/download-route-17-collection-calendars")!)
                    .font(.subheadline)
                Button {
                    showingImporter = true
                } label: {
                    if isUploading {
                        HStack { ProgressView(); Text("Importing calendar…") }
                    } else {
                        FeatureEntryLabel(title: "Choose ICS Calendar", subtitle: "Select a .ics file from Files", systemImage: "calendar.badge.plus")
                    }
                }
                .disabled(isUploading)
                .accessibilityIdentifier("Choose Bin Calendar")
            } header: { Text("Collection schedule") }
            .listRowBackground(AppTheme.surface)

            Section {
                Toggle("Send bin reminders at 7pm", isOn: Binding(
                    get: { settings?.enabled ?? false },
                    set: { enabled in Task { await updateReminders(enabled: enabled) } }
                ))
                .disabled(isUploading || (settings.map { $0.collectionCount == 0 || !$0.pushAvailable } ?? true))
                if settings?.collectionCount ?? 0 > 0 {
                    Text("The reminder goes to each iPhone that has notifications enabled.")
                        .font(.footnote)
                        .foregroundStyle(AppTheme.secondary)
                }
                if settings?.pushAvailable == false {
                    Text("The server needs Apple push credentials before reminders can start.")
                        .font(.footnote)
                        .foregroundStyle(AppTheme.secondary)
                }
                if let statusMessage {
                    Text(statusMessage)
                        .font(.footnote)
                        .foregroundStyle(AppTheme.secondary)
                        .accessibilityIdentifier("Bin Calendar Status")
                }
            } header: { Text("Reminders") }
            .listRowBackground(AppTheme.surface)
        }
        .appListStyle()
        .navigationTitle("Bin collections")
        .navigationBarTitleDisplayMode(.large)
        .fileImporter(isPresented: $showingImporter, allowedContentTypes: [.data], allowsMultipleSelection: false, onCompletion: handleImport)
        .overlay { if isLoading { ProgressView() } }
        .errorAlert($actionError)
        .task { await load() }
    }

    private func handleImport(_ result: Result<[URL], Error>) {
        switch result {
        case .success(let urls):
            guard let url = urls.first else { return }
            guard url.pathExtension.lowercased() == "ics" else {
                actionError = .badRequest(serverMessage: "Select an .ics calendar file.")
                return
            }
            Task { await upload(url) }
        case .failure(let error):
            actionError = .from(error)
        }
    }

    private func load() async {
        isLoading = true
        defer { isLoading = false }
        do { settings = try await apiClient.fetchBinSchedule() }
        catch { actionError = .from(error) }
    }

    private func upload(_ url: URL) async {
        let hasSecurityScope = url.startAccessingSecurityScopedResource()
        defer { if hasSecurityScope { url.stopAccessingSecurityScopedResource() } }
        isUploading = true
        defer { isUploading = false }
        do {
            settings = try await apiClient.uploadBinSchedule(from: url)
            statusMessage = "Calendar imported. \(settings?.collectionCount ?? 0) collection dates loaded."
            actionError = nil
        } catch { actionError = .from(error) }
    }

    private func updateReminders(enabled: Bool) async {
        guard let settings else { return }
        do {
            self.settings = try await apiClient.updateBinReminders(enabled: enabled)
            statusMessage = enabled ? "7pm reminders are on." : "7pm reminders are off."
            actionError = nil
        } catch {
            self.settings = settings
            actionError = .from(error)
        }
    }
}
