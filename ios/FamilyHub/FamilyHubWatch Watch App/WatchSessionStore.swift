import FamilyHubKit
import Foundation
import WatchConnectivity

/// What the link has actually done, so a failure shows on the wrist instead of
/// presenting as a permanently empty screen.
///
/// There is no console on a watch in a kitchen. The first version of this file
/// swallowed every decode failure with `try?`, so a payload the phone happily
/// reported as sent looked identical to one that was never sent at all — which
/// is exactly how the `NSNull` transport bug hid.
struct WatchLinkDiagnostics: Equatable {
    var activation = "not started"
    var handoffsReceived = 0
    var contextsReceived = 0
    var lastError: String?
    var lastEventAt: Date?

    /// A single line short enough for a watch face, listing only what is known.
    var summary: String {
        var parts = ["Link: \(activation)"]
        if handoffsReceived > 0 || contextsReceived > 0 {
            parts.append("\(handoffsReceived) recipe, \(contextsReceived) sync")
        } else {
            parts.append("nothing received")
        }
        if let lastError {
            parts.append(lastError)
        }
        return parts.joined(separator: " · ")
    }
}

/// The watch's half of the link to the phone.
///
/// Receives two things: credentials via the application context, and a recipe to
/// cook via `transferUserInfo`. Both are persisted immediately, so opening the
/// app cold — long after the phone sent anything — still works.
@Observable @MainActor
final class WatchSessionStore: NSObject, WCSessionDelegate {
    static let shared = WatchSessionStore()

    /// The recipe to cook, if one has ever arrived.
    private(set) var handoff: CookHandoff?
    /// Synced from the phone and stored ready for network features. Cook mode
    /// itself never needs it — the whole recipe travels, so the watch cooks
    /// entirely offline.
    private(set) var credentials: WatchCredentials?
    private(set) var diagnostics = WatchLinkDiagnostics()

    private let store: WatchCredentialStore

    init(store: WatchCredentialStore = WatchCredentialStore()) {
        self.store = store
        super.init()
        self.credentials = store.load()
        self.handoff = store.loadLastHandoff()
    }

    func activate() {
        guard WCSession.isSupported() else {
            diagnostics.activation = "unsupported"
            return
        }
        let session = WCSession.default
        session.delegate = self
        session.activate()
        diagnostics.activation = "activating"
    }

    /// Forget the current recipe, returning the app to its empty state.
    func clearHandoff() {
        handoff = nil
        store.clearLastHandoff()
    }

    // MARK: - Receiving

    private func apply(credentialsPayload payload: [String: Any]) {
        diagnostics.contextsReceived += 1
        diagnostics.lastEventAt = Date()
        do {
            let received = try WatchCredentials.fromWatchPayload(payload)
            store.save(received)
            credentials = received.apiToken.isEmpty ? nil : received
        } catch {
            diagnostics.lastError = "sign-in sync failed: \(error)"
        }
    }

    private func apply(handoffPayload payload: [String: Any]) {
        diagnostics.handoffsReceived += 1
        diagnostics.lastEventAt = Date()
        do {
            let received = try CookHandoff.fromWatchPayload(payload)
            // Transfers can arrive out of order after a period offline; keep the
            // newest rather than whichever landed last.
            if let existing = handoff, existing.sentAt > received.sentAt { return }
            store.saveLastHandoff(received)
            handoff = received
            diagnostics.lastError = nil
        } catch {
            diagnostics.lastError = "recipe unreadable: \(error)"
        }
    }

    // MARK: - WCSessionDelegate

    // Delivered on a background queue, so these are nonisolated and hop to the
    // main actor before touching state.

    nonisolated func session(
        _ session: WCSession,
        activationDidCompleteWith activationState: WCSessionActivationState,
        error: Error?
    ) {
        let state = Self.describe(activationState)
        let failure = error?.localizedDescription
        // The context holds whatever the phone last sent, so this is how the
        // watch catches up after being installed or reinstalled.
        let context = session.receivedApplicationContext

        Task { @MainActor in
            self.diagnostics.activation = state
            if let failure { self.diagnostics.lastError = failure }
            guard activationState == .activated, !context.isEmpty else { return }
            self.apply(credentialsPayload: context)
        }
    }

    nonisolated func session(_ session: WCSession, didReceiveApplicationContext applicationContext: [String: Any]) {
        Task { @MainActor in self.apply(credentialsPayload: applicationContext) }
    }

    nonisolated func session(_ session: WCSession, didReceiveUserInfo userInfo: [String: Any] = [:]) {
        Task { @MainActor in self.apply(handoffPayload: userInfo) }
    }

    private nonisolated static func describe(_ state: WCSessionActivationState) -> String {
        switch state {
        case .activated: return "connected"
        case .inactive: return "inactive"
        case .notActivated: return "not connected"
        @unknown default: return "unknown"
        }
    }
}
