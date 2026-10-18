import SwiftUI
import UIKit
import FamilyHubKit

extension Color {
    static let appGreen = Color(.systemGreen)
    static let appRed = Color(.systemRed)
    static let appOrange = Color(.systemOrange)
}

struct GroupedCardModifier: ViewModifier {
    var radius: CGFloat = 16
    @Environment(\.colorSchemeContrast) private var contrast

    func body(content: Content) -> some View {
        content
            .background(Color(.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: radius))
            .overlay {
                if contrast == .increased {
                    RoundedRectangle(cornerRadius: radius)
                        .stroke(Color.primary, lineWidth: 1)
                }
            }
    }
}

struct AdaptiveTabStyle: ViewModifier {
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize

    @ViewBuilder
    func body(content: Content) -> some View {
        if dynamicTypeSize.isAccessibilitySize {
            content.tabViewStyle(.tabBarOnly)
        } else {
            content.tabViewStyle(.sidebarAdaptable)
        }
    }
}

struct AccessibleMotionModifier: ViewModifier {
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    func body(content: Content) -> some View {
        content.transaction { transaction in
            if reduceMotion {
                transaction.animation = nil
                transaction.disablesAnimations = true
            }
        }
    }
}

extension View {
    func groupedBackground() -> some View {
        background(Color(.systemGroupedBackground))
    }

    func groupedCard(radius: CGFloat = 16) -> some View {
        modifier(GroupedCardModifier(radius: radius))
    }
}

extension Color {
    init?(hex: String) {
        guard let parsed = HexColor(hex: hex) else { return nil }
        self.init(red: parsed.red, green: parsed.green, blue: parsed.blue)
    }
}

struct SectionHeaderLabel: View {
    let text: String
    var color: Color = .secondary

    var body: some View {
        Text(text)
            .font(.headline)
            .foregroundStyle(color)
            .frame(maxWidth: .infinity, alignment: .leading)
            .padding(.horizontal, 20)
            .padding(.top, 24)
            .padding(.bottom, 8)
            .accessibilityAddTraits(.isHeader)
    }
}
