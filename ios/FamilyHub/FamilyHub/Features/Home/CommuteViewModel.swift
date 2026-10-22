import Foundation
import Observation
import FamilyHubKit

@Observable
@MainActor
final class CommuteViewModel {
    private(set) var activeDirections: [CommuteDirection] = []
    private(set) var states: [CommuteDirection: ViewState<CommuteBoard>] = [:]

    private let apiClient: any APIClientProtocol
    private let store: CommutePreferencesStore
    private let now: () -> Date
    private var preferences = CommutePreferences()
    private var session = UUID()
    private var userID: String?
    private var attempts: [CommuteDirection: Date] = [:]
    private var requestIDs: [CommuteDirection: UUID] = [:]
    private var requests: [CommuteDirection: Task<Void, Never>] = [:]

    init(apiClient: any APIClientProtocol, store: CommutePreferencesStore? = nil, now: @escaping () -> Date = Date.init) {
        self.apiClient = apiClient
        self.store = store ?? CommutePreferencesStore()
        self.now = now
    }

    func configure(userID: String) {
        stop()
        self.userID = userID
        preferences = store.load(server: apiClient.serverIdentifier ?? "demo", userID: userID)
        tick()
    }

    func run() async {
        stop()
        let identitySession = session
        do {
            let user = try await apiClient.fetchMe()
            guard !Task.isCancelled, session == identitySession else { return }
            configure(userID: user.id)
            let activeSession = session
            defer { if session == activeSession { stop() } }
            while !Task.isCancelled && session == activeSession {
                tick()
                _ = startRequests(force: false)
                try await Task.sleep(for: .seconds(1))
            }
        } catch {
            // A new activation retries identity before it reads local settings.
        }
    }

    func tick() {
        let directions = userID == nil ? [] : preferences.activeDirections(at: now())
        for direction in CommuteDirection.allCases where !directions.contains(direction) {
            requests.removeValue(forKey: direction)?.cancel()
            requestIDs.removeValue(forKey: direction)
            attempts.removeValue(forKey: direction)
            states.removeValue(forKey: direction)
        }
        activeDirections = directions
        for direction in directions {
            if case .loaded(let board) = states[direction], !isFresh(board) {
                states[direction] = .failed(.timedOut)
            }
        }
    }

    func refresh(force: Bool = false) async {
        tick()
        let pending = startRequests(force: force)
        for task in pending { await task.value }
    }

    func retry(_ direction: CommuteDirection) async {
        tick()
        guard activeDirections.contains(direction) else { return }
        attempts.removeValue(forKey: direction)
        let pending = startRequests(force: false)
        for task in pending { await task.value }
    }

    func stop() {
        session = UUID()
        for task in requests.values { task.cancel() }
        requests = [:]
        requestIDs = [:]
        attempts = [:]
        states = [:]
        activeDirections = []
        userID = nil
    }

    private func startRequests(force: Bool) -> [Task<Void, Never>] {
        for direction in activeDirections {
            guard requests[direction] == nil,
                  force || now().timeIntervalSince(attempts[direction] ?? .distantPast) >= 30 else { continue }
            attempts[direction] = now()
            if case .loaded = states[direction] {} else { states[direction] = .loading }
            let requestID = UUID()
            let requestSession = session
            requestIDs[direction] = requestID
            requests[direction] = Task { [weak self] in
                guard let self else { return }
                let result: Result<CommuteBoard, Error>
                do { result = .success(try await apiClient.fetchCommuteTrains(direction: direction, count: 3)) }
                catch { result = .failure(error) }
                guard session == requestSession, requestIDs[direction] == requestID else { return }
                requests.removeValue(forKey: direction)
                requestIDs.removeValue(forKey: direction)
                guard !Task.isCancelled else { return }
                tick()
                guard activeDirections.contains(direction) else { return }
                switch result {
                case .success(let board):
                    guard board.direction == direction else {
                        states[direction] = .failed(.decoding)
                        return
                    }
                    guard isFresh(board) else {
                        states[direction] = .failed(.timedOut)
                        return
                    }
                    attempts[direction] = board.fetchedDate
                    states[direction] = .loaded(board)
                case .failure(let error): states[direction] = .failed(.from(error))
                }
            }
        }
        return activeDirections.compactMap { requests[$0] }
    }

    private func isFresh(_ board: CommuteBoard) -> Bool {
        guard let fetched = board.fetchedDate else { return false }
        let age = now().timeIntervalSince(fetched)
        return age >= -5 && age < 30
    }
}
