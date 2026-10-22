import Foundation

public enum CommuteDirection: String, Codable, CaseIterable, Sendable {
    case toWork = "to_work"
    case toHome = "to_home"

    public var title: String { self == .toWork ? "To work" : "To home" }
}

public struct CommuteSchedule: Codable, Equatable, Sendable {
    public var enabled: Bool
    public var weekdays: Set<Int>
    public var start: String
    public var end: String

    public init(enabled: Bool = false, weekdays: Set<Int> = [2, 3, 4, 5, 6], start: String, end: String) {
        self.enabled = enabled
        self.weekdays = weekdays
        self.start = start
        self.end = end
    }

    public var validationMessage: String? {
        guard !weekdays.isEmpty, weekdays.isSubset(of: Set(1...7)) else { return "Choose at least one day." }
        guard let startMinute = Self.minutes(start), let endMinute = Self.minutes(end), startMinute != endMinute else {
            return "Choose different start and end times."
        }
        return nil
    }

    public func contains(_ date: Date) -> Bool {
        guard enabled, validationMessage == nil,
              let startMinute = Self.minutes(start), let endMinute = Self.minutes(end) else { return false }
        var calendar = Calendar(identifier: .gregorian)
        calendar.timeZone = TimeZone(identifier: "Europe/London")!
        let components = calendar.dateComponents([.weekday, .hour, .minute], from: date)
        let minute = components.hour! * 60 + components.minute!
        let weekday = components.weekday!
        if startMinute < endMinute {
            return weekdays.contains(weekday) && minute >= startMinute && minute < endMinute
        }
        if minute >= startMinute { return weekdays.contains(weekday) }
        let previousWeekday = weekday == 1 ? 7 : weekday - 1
        return minute < endMinute && weekdays.contains(previousWeekday)
    }

    public static func minutes(_ value: String) -> Int? {
        let parts = value.split(separator: ":", omittingEmptySubsequences: false)
        guard parts.count == 2, parts.allSatisfy({ $0.count == 2 && $0.allSatisfy(\.isASCII) && $0.allSatisfy(\.isNumber) }),
              let hour = Int(parts[0]), let minute = Int(parts[1]), (0...23).contains(hour), (0...59).contains(minute) else { return nil }
        return hour * 60 + minute
    }
}

public struct CommutePreferences: Codable, Equatable, Sendable {
    public var toWork: CommuteSchedule
    public var toHome: CommuteSchedule

    public init(toWork: CommuteSchedule = CommuteSchedule(start: "06:30", end: "09:30"),
                toHome: CommuteSchedule = CommuteSchedule(start: "16:00", end: "19:30")) {
        self.toWork = toWork
        self.toHome = toHome
    }

    public subscript(direction: CommuteDirection) -> CommuteSchedule {
        get { direction == .toWork ? toWork : toHome }
        set {
            if direction == .toWork { toWork = newValue } else { toHome = newValue }
        }
    }

    public func activeDirections(at date: Date) -> [CommuteDirection] {
        CommuteDirection.allCases.filter { self[$0].contains(date) }
    }
}

public final class CommutePreferencesStore {
    private let defaults: UserDefaults

    public init(defaults: UserDefaults = .standard) {
        self.defaults = defaults
    }

    public func load(server: String, userID: String) -> CommutePreferences {
        guard let data = defaults.data(forKey: key(server: server, userID: userID)),
              let preferences = try? JSONDecoder().decode(CommutePreferences.self, from: data),
              preferences.toWork.validationMessage == nil, preferences.toHome.validationMessage == nil else {
            return CommutePreferences()
        }
        return preferences
    }

    @discardableResult
    public func save(_ preferences: CommutePreferences, server: String, userID: String) -> Bool {
        guard !userID.isEmpty, preferences.toWork.validationMessage == nil, preferences.toHome.validationMessage == nil,
              let data = try? JSONEncoder().encode(preferences) else { return false }
        defaults.set(data, forKey: key(server: server, userID: userID))
        return true
    }

    private func key(server: String, userID: String) -> String {
        var components = URLComponents(string: server)
        let host = components?.host?.lowercased()
        let scheme = components?.scheme?.lowercased()
        components?.host = host
        components?.scheme = scheme
        if (components?.scheme == "https" && components?.port == 443) || (components?.scheme == "http" && components?.port == 80) {
            components?.port = nil
        }
        while components?.path.hasSuffix("/") == true { components?.path.removeLast() }
        let identity = [components?.string ?? server, userID]
        let suffix = (try? JSONEncoder().encode(identity))?.base64EncodedString() ?? ""
        return "commute.preferences.\(suffix)"
    }
}
