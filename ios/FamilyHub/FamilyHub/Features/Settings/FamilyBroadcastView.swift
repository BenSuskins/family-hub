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
    @State private var showingResult = false
    @State private var showingNotifications = false
    @State private var actionError: APIError?
    @AccessibilityFocusState private var resultIsFocused: Bool

    private var trimmedMessage: String { message.trimmingCharacters(in: .whitespacesAndNewlines) }

    var body: some View {
        Form {
            Section {
                VStack(alignment: .leading, spacing: 12) {
                    Text("Send a family message").font(.system(.largeTitle, design: .rounded, weight: .bold))
                        .accessibilityAddTraits(.isHeader)
                    Text("Send a notification to registered family iPhones.")
                        .font(.subheadline).foregroundStyle(AppTheme.secondary)
                }
            }.listRowBackground(Color.clear)
            Section {
                TextField("Message", text: $message, axis: .vertical)
                    .lineLimit(4...10)
                    .accessibilityLabel("Family message")
                    .focused($messageIsFocused)
            } header: { Text("Message") } footer: {
                Text("\(message.unicodeScalars.count)/500 characters")
            }.listRowBackground(AppTheme.surface).disabled(isSending)
            Section {
                Text("Only iPhones with notifications enabled receive this message. It can appear on the lock screen.")
                    .font(.subheadline).foregroundStyle(AppTheme.secondary)
            }.listRowBackground(AppTheme.sage)
            if !isLoading && !pushAvailable {
                Section {
                    Text("The server needs Apple push credentials before messages can send.")
                        .foregroundStyle(AppTheme.secondary)
                }.listRowBackground(AppTheme.surface)
            }
            Section {
                Button(isSending ? "Sending…" : "Send to family") { Task { await send() } }
                    .buttonStyle(AppPrimaryButtonStyle())
                    .accessibilityIdentifier("Send to Family")
                    .disabled(isLoading || isSending || !pushAvailable || trimmedMessage.isEmpty || trimmedMessage.unicodeScalars.count > 500)
            }.listRowBackground(Color.clear).listRowInsets(EdgeInsets())
        }
        .appListStyle()
        .navigationTitle("Family message")
        .navigationBarTitleDisplayMode(.inline)
        .navigationDestination(isPresented: $showingResult) {
            if let result {
                Form {
                    Section {
                        Text(result.accepted == 0 ? "Message not sent" : "Message sent")
                            .font(.system(.largeTitle, design: .rounded, weight: .bold))
                            .accessibilityAddTraits(.isHeader)
                    }.listRowBackground(Color.clear)
                    Section {
                        VStack(alignment: .leading, spacing: 12) {
                            Image(systemName: result.accepted == 0 ? "bell.slash" : "checkmark.circle")
                                .font(.title).accessibilityHidden(true)
                            if result.accepted == 0 && result.failed == 0 {
                                Text("No available iPhones").font(.headline).accessibilityFocused($resultIsFocused)
                                Text("No family iPhones have notifications enabled.").foregroundStyle(AppTheme.secondary)
                            } else {
                                Text("Apple accepted: \(result.accepted)").font(.headline).accessibilityFocused($resultIsFocused)
                                if result.failed > 0 {
                                    Text("Failed or pending: \(result.failed)")
                                    Text("Return to the message and send again to retry eligible failures. Accepted notifications do not repeat.")
                                        .font(.subheadline).foregroundStyle(AppTheme.secondary)
                                }
                            }
                        }
                    }.listRowBackground(AppTheme.surface)
                    Section("Your message") { Text(message) }.listRowBackground(AppTheme.surface)
                    Section {
                        Button("Back to message") { showingResult = false }
                            .buttonStyle(AppPrimaryButtonStyle())
                    }.listRowBackground(Color.clear).listRowInsets(EdgeInsets())
                    Section {
                        Button("Notification help") { showingNotifications = true }
                            .buttonStyle(AppSecondaryButtonStyle())
                    }.listRowBackground(Color.clear).listRowInsets(EdgeInsets())
                }
                .appListStyle()
                .navigationTitle("Family message")
                .navigationBarTitleDisplayMode(.inline)
                .navigationDestination(isPresented: $showingNotifications) {
                    ChoreRemindersView(apiClient: apiClient, isAdmin: true)
                }
            }
        }
        .toolbar {
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
            showingResult = true
            resultIsFocused = true
        }
        catch { actionError = .from(error) }
    }
}
