import XCTest
import SwiftUI
import UIKit
@testable import FamilyHub

@MainActor
final class AppThemeAccessibilityTests: XCTestCase {
    func testBodyAndActionTextMeetContrastRequirementsInBothAppearances() {
        for appearance in [UIUserInterfaceStyle.light, .dark] {
            let traits = UITraitCollection(userInterfaceStyle: appearance)
            for background in [AppTheme.background, AppTheme.surface] {
                for foreground in [AppTheme.ink, AppTheme.secondary, AppTheme.accent, AppTheme.destructive, AppTheme.clay] {
                    XCTAssertGreaterThanOrEqual(contrast(foreground, background, traits: traits), 4.5)
                }
            }
            XCTAssertGreaterThanOrEqual(contrast(AppTheme.onAccent, AppTheme.accent, traits: traits), 4.5)
        }
    }

    private func contrast(_ foreground: Color, _ background: Color, traits: UITraitCollection) -> Double {
        let values = [luminance(foreground, traits: traits), luminance(background, traits: traits)]
        return (values.max()! + 0.05) / (values.min()! + 0.05)
    }

    private func luminance(_ color: Color, traits: UITraitCollection) -> Double {
        let resolved = UIColor(color).resolvedColor(with: traits)
        var red: CGFloat = 0
        var green: CGFloat = 0
        var blue: CGFloat = 0
        var alpha: CGFloat = 0
        XCTAssertTrue(resolved.getRed(&red, green: &green, blue: &blue, alpha: &alpha))
        func linear(_ component: CGFloat) -> Double {
            let value = Double(component)
            return value <= 0.04045 ? value / 12.92 : pow((value + 0.055) / 1.055, 2.4)
        }
        return 0.2126 * linear(red) + 0.7152 * linear(green) + 0.0722 * linear(blue)
    }
}
