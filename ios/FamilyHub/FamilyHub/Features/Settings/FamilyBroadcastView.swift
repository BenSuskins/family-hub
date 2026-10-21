import SwiftUI
import FamilyHubKit

struct FamilyBroadcastView: View {
    let apiClient: any APIClientProtocol
    @State private var message = ""
    @FocusState private var messageIsFocused: Bool
    @State private var requestID = UUID().uuidString
    @State private var isSending = false
    @State private var isLoading = true
    @State private var pushAvailable = false
    @State private var result: BroadcastResult?
    @State private var actionError: APIError?
    @AccessibilityFocusState private var resultIsFocused: Bool

    private var trimmedMessage: String { message.trimmingCharacters(in: .whitespacesAndNewlines) }

    var body: some View {
        Form {
            Section {
                TextField("Message", text: $message, axis: .vertical)
                    .lineLimit(4...10)
                    .accessibilityLabel("Family message")
                    .focused($messageIsFocused)
                Text("\(message.unicodeScalars.count)/500 characters")
                    .font(.caption)
                    .foregroundStyle(AppTheme.secondary)
            } header: {
                Text("Family Message")
            } footer: {
                Text("Send a notification to every family iPhone with notifications enabled. The message can appear on the lock screen.")
            }.listRowBackground(AppTheme.surface)
            .disabled(isSending)

            if !isLoading && !pushAvailable {
                Section {
                    Text("The server needs Apple push credentials before messages can send.")
                        .foregroundStyle(AppTheme.secondary)
                }.listRowBackground(AppTheme.surface)
            }
            if let result {
                Section("Send result") {
                    if result.accepted == 0 && result.failed == 0 {
                        Text("No available iPhones").font(.headline).accessibilityFocused($resultIsFocused)
                        Text("No family iPhones have notifications enabled.").foregroundStyle(AppTheme.secondary)
                        NavigationLink("Set up notifications") { ChoreRemindersView(apiClient: apiClient, isAdmin: true) }
                    } else {
                        Text("Apple accepted: \(result.accepted)").accessibilityFocused($resultIsFocused)
                    }
                    if result.failed > 0 {
                        Text("Failed or pending: \(result.failed)")
                        Text("Tap Send again to retry eligible failures. Accepted notifications do not repeat.")
                            .foregroundStyle(AppTheme.secondary)
                    }
                }.listRowBackground(AppTheme.surface)
            }
        }
        .appListStyle()
        .navigationTitle("Family message")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .confirmationAction) {
                if isSending { ProgressView() }
                else {
                    Button("Send") { Task { await send() } }.buttonStyle(AppCommitButtonStyle())
                        .accessibilityLabel("Send to Family")
                        .disabled(isLoading || !pushAvailable || trimmedMessage.isEmpty || trimmedMessage.unicodeScalars.count > 500)
                }
            }
            ToolbarItemGroup(placement: .keyboard) {
                Spacer()
                Button("Done") { messageIsFocused = false }
            }
        }
        .errorAlert($actionError)
        .task {
            defer { isLoading = false }
            do { pushAvailable = try await apiClient.fetchChoreReminders().pushAvailable }
            catch { actionError = .from(error) }
        }
        .onChange(of: message) { _, _ in
            requestID = UUID().uuidString
            result = nil
        }
    }

    private func send() async {
        messageIsFocused = false
        isSending = true
        defer { isSending = false }
        do {
            result = try await apiClient.sendFamilyBroadcast(message: trimmedMessage, requestID: requestID)
            resultIsFocused = true
        }
        catch { actionError = .from(error) }
    }
}
