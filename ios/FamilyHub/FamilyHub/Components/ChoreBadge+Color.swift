import SwiftUI
import FamilyHubKit

// ChoreBadge lives in FamilyHubKit, which is deliberately Foundation-only so it
// also builds for watchOS and for the Linux CI job. The badge's tint is the one
// piece of it that is presentation, so it stays here in the app.
extension ChoreBadge {
    var color: Color {
        switch self {
        case .overdue:            return .red
        case .dueToday, .dueSoon: return .orange
        }
    }
}
