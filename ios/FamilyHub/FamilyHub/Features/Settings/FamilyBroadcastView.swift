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
                    .foregroundStyle(.secondary)
            } header: {
                Text("Family Message")
            } footer: {
                Text("Send a notification to every family iPhone with notifications enabled. The message can appear on the lock screen.")
            }
            .disabled(isSending)

            if !isLoading && !pushAvailable {
                Section {
                    Text("The server needs Apple push credentials before messages can send.")
                        .foregroundStyle(.secondary)
                }
            }
            if let result {
                Section("Send Result") {
                    Text("Apple accepted: \(result.accepted)")
                    if result.failed > 0 {
                        Text("Failed or pending: \(result.failed)")
                        Text("Tap Send again to retry eligible failures. Accepted notifications do not repeat.")
                            .foregroundStyle(.secondary)
                    } else if result.accepted == 0 {
                        Text("No family iPhones have notifications enabled.")
                            .foregroundStyle(.secondary)
                    }
                }
            }
        }
        .navigationTitle("Family Broadcast")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .confirmationAction) {
                if isSending { ProgressView() }
                else {
                    Button("Send") { Task { await send() } }
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
        do { result = try await apiClient.sendFamilyBroadcast(message: trimmedMessage, requestID: requestID) }
        catch { actionError = .from(error) }
    }
}
