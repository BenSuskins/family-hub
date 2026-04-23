import XCTest
@testable import FamilyHub

final class UserTests: XCTestCase {
    func testDecodesFromJSON() throws {
        let json = """
        {"ID":"u1","Name":"Ben Suskins","Email":"ben@example.com","AvatarURL":"","Role":"admin"}
        """.data(using: .utf8)!
        let user = try JSONDecoder().decode(User.self, from: json)
        XCTAssertEqual(user.id, "u1")
        XCTAssertEqual(user.name, "Ben Suskins")
    }

    func testInitialsFromTwoWordName() {
        let user = User(id: "u1", name: "Ben Suskins", email: "", avatarURL: "", role: "member")
        XCTAssertEqual(user.initials, "BS")
    }

    func testInitialsFromSingleWordName() {
        let user = User(id: "u2", name: "Admin", email: "", avatarURL: "", role: "member")
        XCTAssertEqual(user.initials, "A")
    }

    func testInitialsFromEmptyName() {
        let user = User(id: "u3", name: "", email: "", avatarURL: "", role: "member")
        XCTAssertEqual(user.initials, "?")
    }
}
