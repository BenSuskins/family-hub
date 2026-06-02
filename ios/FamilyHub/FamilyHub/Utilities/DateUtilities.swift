import Foundation

/// Shared, cached formatters and parsing for the machine-readable date formats
/// the API speaks (day keys, month queries, due times, ISO timestamps).
///
/// `DateFormatter` is expensive to create, so these are built once and reused.
/// All fixed formats use the POSIX locale so parsing/formatting is stable and
/// locale-independent — these are wire formats, never user-facing copy. For
/// human-facing display, views keep their own presentation formatters.
enum APIDate {
    /// `yyyy-MM-dd` — day keys for meals, calendar and chore due dates.
    static let day = fixed("yyyy-MM-dd")
    /// `yyyy-MM` — the calendar's month query.
    static let month = fixed("yyyy-MM")
    /// `HH:mm` — chore due times.
    static let time = fixed("HH:mm")
    /// RFC3339 / ISO8601 timestamps, as returned for chore due dates.
    static let iso = ISO8601DateFormatter()

    static func dayString(_ date: Date) -> String { day.string(from: date) }
    static func monthString(_ date: Date) -> String { month.string(from: date) }

    /// Parse an API date string that may be a full ISO8601 timestamp or a short
    /// `yyyy-MM-dd` date (only the day component is needed for the latter).
    static func parse(_ value: String?) -> Date? {
        guard let value, !value.isEmpty else { return nil }
        return iso.date(from: value) ?? day.date(from: String(value.prefix(10)))
    }

    private static func fixed(_ format: String) -> DateFormatter {
        let formatter = DateFormatter()
        formatter.dateFormat = format
        formatter.locale = Locale(identifier: "en_US_POSIX")
        return formatter
    }
}
