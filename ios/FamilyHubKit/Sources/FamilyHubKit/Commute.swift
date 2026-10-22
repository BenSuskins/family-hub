import Foundation

public struct CommuteStation: Codable, Equatable, Sendable {
    public let code: String

    public init(code: String) {
        self.code = code
    }
}

public struct CommuteBoard: Codable, Equatable, Sendable {
    public let origin: CommuteStation
    public let destination: CommuteStation
    public let generatedAt: String
    public let fetchedAt: String
    public let timezone: String
    public let trains: [CommuteTrain]

    public var fetchedDate: Date? {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        if let date = formatter.date(from: fetchedAt) { return date }
        formatter.formatOptions = [.withInternetDateTime]
        return formatter.date(from: fetchedAt)
    }

    public init(origin: CommuteStation, destination: CommuteStation,
                generatedAt: String, fetchedAt: String, timezone: String = "Europe/London", trains: [CommuteTrain]) {
        self.origin = origin
        self.destination = destination
        self.generatedAt = generatedAt
        self.fetchedAt = fetchedAt
        self.timezone = timezone
        self.trains = trains
    }
}

public struct CommuteTime: Equatable, Sendable {
    public let scheduled: String?
    public let expected: String?

    public init(scheduled: String?, expected: String?) {
        self.scheduled = scheduled.flatMap { CommuteSchedule.minutes($0) == nil ? nil : $0 }
        self.expected = expected.flatMap { CommuteSchedule.minutes($0) == nil ? nil : $0 }
    }

    public var display: String { expected ?? scheduled ?? "Time unavailable" }
    public var original: String? {
        guard let scheduled, let expected, scheduled != expected else { return nil }
        return scheduled
    }

    public func accessibilityLabel(_ name: String) -> String {
        if let original { return "Scheduled \(name) \(original), expected \(name) \(display)" }
        if expected != nil { return "\(name.capitalized) \(display)" }
        if scheduled != nil { return "Scheduled \(name) \(display), estimate unavailable" }
        return "\(name.capitalized) time unavailable"
    }
}

public struct CommuteTrain: Codable, Equatable, Identifiable, Sendable {
    public let serviceID: String
    public let operatorName: String
    public let platform: String?
    public let status: String
    public let scheduledDeparture: String?
    public let expectedDeparture: String?
    public let scheduledArrival: String?
    public let expectedArrival: String?
    public let cancellationReason: String?
    public let delayReason: String?

    public var id: String { serviceID }
    public var isCancelled: Bool { status == "cancelled" }
    public var departure: CommuteTime {
        CommuteTime(scheduled: scheduledDeparture, expected: isCancelled ? nil : expectedDeparture)
    }
    public var arrival: CommuteTime {
        CommuteTime(scheduled: scheduledArrival, expected: isCancelled ? nil : expectedArrival)
    }
    public var durationMinutes: Int? {
        guard !isCancelled, let departure = expectedDeparture.flatMap(CommuteSchedule.minutes),
              let arrival = expectedArrival.flatMap(CommuteSchedule.minutes) else { return nil }
        return (arrival - departure + 1440) % 1440
    }
    public var statusLabel: String {
        switch status {
        case "cancelled": return "Cancelled"
        case "delayed": return "Delayed"
        case "on_time": return "On time"
        default: return "Status unknown"
        }
    }

    public init(serviceID: String, operatorName: String = "", platform: String? = nil, status: String,
                scheduledDeparture: String? = nil, expectedDeparture: String? = nil,
                scheduledArrival: String? = nil, expectedArrival: String? = nil,
                cancellationReason: String? = nil, delayReason: String? = nil) {
        self.serviceID = serviceID
        self.operatorName = operatorName
        self.platform = platform
        self.status = status
        self.scheduledDeparture = scheduledDeparture
        self.expectedDeparture = expectedDeparture
        self.scheduledArrival = scheduledArrival
        self.expectedArrival = expectedArrival
        self.cancellationReason = cancellationReason
        self.delayReason = delayReason
    }

    enum CodingKeys: String, CodingKey {
        case serviceID = "serviceId"
        case operatorName = "operator"
        case platform, status, scheduledDeparture, expectedDeparture, scheduledArrival, expectedArrival
        case cancellationReason, delayReason
    }
}
