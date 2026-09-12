import FamilyHubKit
import Foundation
import WatchConnectivity

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

    private let store: WatchCredentialStore

    init(store: WatchCredentialStore = WatchCredentialStore()) {
        self.store = store
        super.init()
        self.credentials = store.load()
        self.handoff = store.loadLastHandoff()
    }

    func activate() {
        guard WCSession.isSupported() else { return }
        let session = WCSession.default
        session.delegate = self
        session.activate()
    }

    /// Forget the current recipe, returning the app to its empty state.
    func clearHandoff() {
        handoff = nil
        store.clearLastHandoff()
    }

    // MARK: - Receiving

    private func apply(credentialsPayload payload: [String: Any]) {
        guard let received = try? WatchCredentials.fromWatchPayload(payload) else { return }
        store.save(received)
        credentials = received.apiToken.isEmpty ? nil : received
    }

    private func apply(handoffPayload payload: [String: Any]) {
        guard let received = try? CookHandoff.fromWatchPayload(payload) else { return }
        // Transfers can arrive out of order after a period offline; keep the
        // newest rather than whichever landed last.
        if let existing = handoff, existing.sentAt > received.sentAt { return }
        store.saveLastHandoff(received)
        handoff = received
    }

    // MARK: - WCSessionDelegate

    // Delivered on a background queue, so these are nonisolated and hop to the
    // main actor before touching state.

    nonisolated func session(
        _ session: WCSession,
        activationDidCompleteWith activationState: WCSessionActivationState,
        error: Error?
    ) {
        guard activationState == .activated else { return }
        // The context holds whatever the phone last sent, so this is how the
        // watch catches up after being installed or reinstalled.
        let context = session.receivedApplicationContext
        guard !context.isEmpty else { return }
        Task { @MainActor in self.apply(credentialsPayload: context) }
    }

    nonisolated func session(_ session: WCSession, didReceiveApplicationContext applicationContext: [String: Any]) {
        Task { @MainActor in self.apply(credentialsPayload: applicationContext) }
    }

    nonisolated func session(_ session: WCSession, didReceiveUserInfo userInfo: [String: Any] = [:]) {
        Task { @MainActor in self.apply(handoffPayload: userInfo) }
    }
}
