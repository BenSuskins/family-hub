import SwiftUI
import UniformTypeIdentifiers
import FamilyHubKit

@Observable
final class DatabaseRestoreSelection {
    private(set) var selectedFileURL: URL?
    var actionError: APIError?

    var requiresConfirmation: Bool { selectedFileURL != nil }

    @discardableResult
    func select(_ url: URL) -> Bool {
        guard url.lastPathComponent.lowercased().hasSuffix(".db.gz") else {
            actionError = .badRequest(serverMessage: "Select a .db.gz backup file.")
            return false
        }
        selectedFileURL = url
        actionError = nil
        return true
    }

    func clear() {
        selectedFileURL = nil
    }
}

struct DatabaseBackupView: View {
    let apiClient: any APIClientProtocol

    @State private var backupURL: URL?
    @State private var restoreSelection = DatabaseRestoreSelection()
    @State private var isDownloading = false
    @State private var isRestoring = false
    @State private var showingImporter = false
    @State private var actionError: APIError?
    @State private var statusMessage: String?

    var body: some View {
        List {
            Section {
                Text("Save a copy of the family database or restore it from a backup file.")
                    .font(.subheadline)
                    .foregroundStyle(AppTheme.secondary)
            }.listRowBackground(Color.clear)

            Section("Create a backup") {
                if let backupURL {
                    ShareLink(item: backupURL) {
                        FeatureEntryLabel(title: "Save or Share Backup", subtitle: backupURL.lastPathComponent, systemImage: "square.and.arrow.up")
                    }
                    .accessibilityIdentifier("Save Database Backup")
                }
                Button {
                    Task { await downloadBackup() }
                } label: {
                    if isDownloading {
                        HStack { ProgressView(); Text("Creating backup…") }
                    } else {
                        FeatureEntryLabel(title: "Create Backup", subtitle: "Download a .db.gz database archive", systemImage: "archivebox")
                    }
                }
                .disabled(isDownloading)
                .accessibilityIdentifier("Create Database Backup")
            }.listRowBackground(AppTheme.surface)

            Section("Restore a backup") {
                Button {
                    showingImporter = true
                } label: {
                    if isRestoring {
                        HStack { ProgressView(); Text("Restoring database…") }
                    } else {
                        FeatureEntryLabel(title: "Choose Backup File", subtitle: "Select a .db.gz archive from Files", systemImage: "arrow.down.doc")
                    }
                }
                .disabled(isRestoring)
                .accessibilityIdentifier("Choose Database Backup")

                if let statusMessage {
                    Text(statusMessage).font(.footnote).foregroundStyle(AppTheme.secondary)
                        .accessibilityIdentifier("Database Backup Status")
                }
            }.listRowBackground(AppTheme.surface)
        }
        .appListStyle()
        .navigationTitle("Database Backup")
        .navigationBarTitleDisplayMode(.large)
        .fileImporter(isPresented: $showingImporter, allowedContentTypes: [.data], allowsMultipleSelection: false, onCompletion: handleImport)
        .confirmationDialog("Replace the database?", isPresented: Binding(
            get: { restoreSelection.requiresConfirmation },
            set: { if !$0 { restoreSelection.clear() } }
        ), titleVisibility: .visible) {
            Button("Restore Database", role: .destructive) { Task { await restoreBackup() } }
            Button("Cancel", role: .cancel) { restoreSelection.clear() }
        } message: {
            Text("This replaces all current family data and restarts the server. Continue only if this is the correct backup.")
        }
        .errorAlert($actionError)
    }

    private func downloadBackup() async {
        isDownloading = true
        defer { isDownloading = false }
        do {
            backupURL = try await apiClient.downloadDatabaseBackup()
            statusMessage = "Backup is ready to save."
            actionError = nil
        } catch {
            actionError = .from(error)
        }
    }

    private func handleImport(_ result: Result<[URL], Error>) {
        switch result {
        case .success(let urls):
            guard let url = urls.first else { return }
            guard restoreSelection.select(url) else {
                actionError = restoreSelection.actionError
                return
            }
            statusMessage = nil
        case .failure(let error):
            actionError = .from(error)
        }
    }

    private func restoreBackup() async {
        guard let restoreURL = restoreSelection.selectedFileURL else { return }
        isRestoring = true
        defer {
            isRestoring = false
            restoreSelection.clear()
        }
        let hasSecurityScope = restoreURL.startAccessingSecurityScopedResource()
        defer { if hasSecurityScope { restoreURL.stopAccessingSecurityScopedResource() } }
        do {
            try await apiClient.restoreDatabaseBackup(from: restoreURL)
            statusMessage = "Restore sent. The server is restarting."
            actionError = nil
        } catch {
            actionError = .from(error)
        }
    }
}
