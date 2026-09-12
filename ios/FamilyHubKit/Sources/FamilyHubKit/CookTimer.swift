import Foundation

/// A running step timer.
///
/// The countdown is derived from `firesAt` rather than ticked down in a stored
/// property, so it stays correct across the thing watchOS does constantly:
/// suspending the app when your wrist drops. On wake the remaining time is
/// recomputed from the clock instead of resuming from a stale value.
public struct CookTimer: Equatable, Sendable {
    /// The step this timer belongs to, so the UI can show it in place.
    public let stepIndex: Int
    public let duration: Int
    public let firesAt: Date

    public init(stepIndex: Int, duration: Int, startedAt: Date = Date()) {
        self.stepIndex = stepIndex
        self.duration = duration
        self.firesAt = startedAt.addingTimeInterval(TimeInterval(duration))
    }

    /// Seconds left, floored at zero.
    public func remaining(at now: Date = Date()) -> Int {
        max(0, Int(firesAt.timeIntervalSince(now).rounded(.up)))
    }

    public func hasFired(at now: Date = Date()) -> Bool {
        remaining(at: now) == 0
    }

    /// Fraction elapsed, 0...1, for a progress ring.
    public func progress(at now: Date = Date()) -> Double {
        guard duration > 0 else { return 1 }
        return min(1, max(0, 1 - Double(remaining(at: now)) / Double(duration)))
    }

    /// The notification identifier, unique per step so a restarted timer
    /// replaces its predecessor rather than firing twice.
    public var notificationID: String { "cook-step-\(stepIndex)" }
}

/// Formats a countdown for a small screen: `9:59`, or `1:05:00` past an hour.
/// Not localised on purpose — these are clock digits, not prose.
public func formatCountdown(_ seconds: Int) -> String {
    let clamped = max(0, seconds)
    let hours = clamped / 3600
    let minutes = (clamped % 3600) / 60
    let secs = clamped % 60
    if hours > 0 {
        return String(format: "%d:%02d:%02d", hours, minutes, secs)
    }
    return String(format: "%d:%02d", minutes, secs)
}

/// Describes a step's timer in words, for the button that starts it.
public func describeDuration(_ seconds: Int) -> String {
    let minutes = seconds / 60
    switch seconds {
    case ..<60:
        return "\(seconds) sec"
    case 60..<3600 where seconds % 60 == 0:
        return "\(minutes) min"
    case 60..<3600:
        return "\(minutes) min \(seconds % 60) sec"
    default:
        let hours = seconds / 3600
        let leftover = (seconds % 3600) / 60
        return leftover == 0 ? "\(hours) hr" : "\(hours) hr \(leftover) min"
    }
}
