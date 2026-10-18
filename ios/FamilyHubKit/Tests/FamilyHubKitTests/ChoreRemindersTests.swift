import XCTest
@testable import FamilyHubKit

final class ChoreRemindersTests: XCTestCase {
    func testSettingsDecodeTheServerContract() throws {
        let data = Data(#"{"enabled":false,"times":["10:00","16:30"],"timezone":"Europe/London","pushAvailable":true}"#.utf8)
        let settings = try JSONDecoder().decode(ChoreReminders.self, from: data)
        XCTAssertEqual(settings.times, ["10:00", "16:30"])
        XCTAssertEqual(settings.timezone, "Europe/London")
        XCTAssertFalse(settings.enabled)
        XCTAssertTrue(settings.pushAvailable)
    }
}
