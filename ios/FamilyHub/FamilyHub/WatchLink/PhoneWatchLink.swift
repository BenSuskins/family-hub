import FamilyHubKit
import Foundation
import WatchConnectivity

/// The phone's half of the link to the watch app.
///
/// Two different jobs, deliberately using two different WatchConnectivity
/// mechanisms:
///
/// * **Credentials** go by `updateApplicationContext`. It keeps only the latest
///   value, delivers in the background, and re-delivers on activation — exactly
///   "here is the current token", and harmless to send repeatedly.
/// * **A recipe** goes by `transferUserInfo`. It queues and is guaranteed, so a
///   handoff survives the phone being asleep or out of range. `sendMessage`
///   would need both apps awake and reachable at that instant.
///
/// The watch cannot get credentials any other way: signing in needs
/// `ASWebAuthenticationSession`, which does not exist on watchOS, and neither the
/// keychain nor the app group crosses the device boundary.
@Observable @MainActor
final class PhoneWatchLink: NSObject, WCSessionDelegate {
    static let shared = PhoneWatchLink()

    /// Whether a "Cook on Watch" action is worth showing at all.
    private(set) var canSendToWatch = false

    /// Why the last handoff did not arrive, if it did not.
    ///
    /// `transferUserInfo` returns immediately and reports failure later on the
    /// delegate, so a send that looks successful can still be dropped in
    /// transit. Without this the phone said "sent" and the watch showed its
    /// empty screen, with nothing anywhere to say which of the two was wrong.
    private(set) var lastTransferError: String?

    private var session: WCSession? {
        WCSession.isSupported() ? WCSession.default : nil
    }

    /// Call once at launch. Safe on devices with no watch — `isSupported()`
    /// is false on iPad, and pairing state simply stays false.
    func activate() {
        guard let session else { return }
        session.delegate = self
        session.activate()
    }

    // MARK: - Credentials

    func sendCredentials(apiToken: String, baseURL: String) {
        push(WatchCredentials(apiToken: apiToken, baseURL: baseURL))
    }

    /// Sign-out: overwrite the context with empty credentials rather than
    /// leaving the last good token sitting on the watch.
    func clearCredentials() {
        push(WatchCredentials(apiToken: "", baseURL: ""))
    }

    /// Re-send whatever is currently stored. Used on activation, and after the
    /// server URL changes, so the watch recovers without a fresh sign-in.
    ///
    /// Reads the app-group mirror that `KeychainStore.saveAPIToken` maintains —
    /// the keychain itself is not readable from every context, and this value is
    /// always written alongside it.
    func syncStoredCredentials() {
        let defaults = UserDefaults(suiteName: "group.uk.co.suskins.familyhub")
        guard let token = defaults?.string(forKey: "api_token"), !token.isEmpty,
              let baseURL = defaults?.string(forKey: "baseURL"), !baseURL.isEmpty else {
            return
        }
        sendCredentials(apiToken: token, baseURL: baseURL)
    }

    private func push(_ credentials: WatchCredentials) {
        guard let session, session.activationState == .activated else { return }
        do {
            try session.updateApplicationContext(try credentials.watchPayload())
        } catch {
            // Not worth surfacing: the context is re-sent on every activation,
            // so a transient failure heals itself next time the watch wakes.
            print("PhoneWatchLink: could not update application context — \(error)")
        }
    }

    // MARK: - Recipe handoff

    /// Hand a recipe to the watch to cook. Returns false when there is nothing
    /// to send to, so the caller can keep the button hidden.
    @discardableResult
    func send(recipe: Recipe) -> Bool {
        guard let session, session.activationState == .activated,
              session.isPaired, session.isWatchAppInstalled else {
            return false
        }
        lastTransferError = nil
        do {
            let payload = try CookHandoff(recipe: recipe).watchPayload()
            // WCSession raises rather than throwing when handed a value that is
            // not a property list, so check before handing it over — a crash in
            // the kitchen is a worse bug than a failed handoff.
            guard isWatchTransportable(payload) else {
                lastTransferError = "This recipe could not be packaged for the watch."
                return false
            }
            _ = session.transferUserInfo(payload)
            return true
        } catch {
            lastTransferError = "Could not package the recipe: \(error.localizedDescription)"
            return false
        }
    }

    // MARK: - WCSessionDelegate

    // These arrive on a background queue, so they are nonisolated and hop back
    // to the main actor before touching any state.

    nonisolated func session(
        _ session: WCSession,
        activationDidCompleteWith activationState: WCSessionActivationState,
        error: Error?
    ) {
        Task { @MainActor in
            self.refreshAvailability()
            if activationState == .activated {
                self.syncStoredCredentials()
            }
        }
    }

    nonisolated func sessionDidBecomeInactive(_ session: WCSession) {}

    nonisolated func sessionDidDeactivate(_ session: WCSession) {
        // Reactivate so the link survives the user switching watches.
        session.activate()
    }

    /// Reports the outcome of a queued `transferUserInfo`, which is the only
    /// place a dropped handoff is ever mentioned.
    nonisolated func session(
        _ session: WCSession,
        didFinish userInfoTransfer: WCSessionUserInfoTransfer,
        error: Error?
    ) {
        guard let error else { return }
        let message = error.localizedDescription
        Task { @MainActor in
            self.lastTransferError = "The watch did not receive it: \(message)"
        }
    }

    nonisolated func sessionWatchStateDidChange(_ session: WCSession) {
        Task { @MainActor in
            self.refreshAvailability()
            self.syncStoredCredentials()
        }
    }

    private func refreshAvailability() {
        guard let session else {
            canSendToWatch = false
            return
        }
        canSendToWatch = session.isPaired && session.isWatchAppInstalled
    }
}
