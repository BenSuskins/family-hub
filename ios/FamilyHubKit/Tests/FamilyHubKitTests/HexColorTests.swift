import XCTest
@testable import FamilyHubKit

final class HexColorTests: XCTestCase {

    func testParsesASixDigitHexWithOrWithoutAHash() {
        XCTAssertEqual(HexColor(hex: "FF8800"), HexColor(hex: "#ff8800"))
    }

    func testComponentsAreScaledToZeroThroughOne() throws {
        let colour = try XCTUnwrap(HexColor(hex: "#3B82F6"))

        XCTAssertEqual(colour.red, Double(0x3B) / 255, accuracy: 0.0001)
        XCTAssertEqual(colour.green, Double(0x82) / 255, accuracy: 0.0001)
        XCTAssertEqual(colour.blue, Double(0xF6) / 255, accuracy: 0.0001)
    }

    func testSurroundingWhitespaceIsIgnored() {
        XCTAssertEqual(HexColor(hex: "  #000000\n"), HexColor(hex: "000000"))
    }

    // Anything unparseable has to be nil rather than black: callers fall back
    // to their own tint, and a row drawn black would look deliberate.
    func testUnparseableValuesAreRejected() {
        for value in ["", "   ", "#FFF", "FF88000", "nottrue", "#GGGGGG", "blue"] {
            XCTAssertNil(HexColor(hex: value), "expected nil for \(value)")
        }
    }
}
