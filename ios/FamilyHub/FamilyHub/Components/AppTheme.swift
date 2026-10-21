import SwiftUI
import UIKit
import FamilyHubKit


enum AppTheme {
    static func configureNavigationAppearance() {
        let base = UIFont.systemFont(ofSize: 34, weight: .bold)
        let descriptor = base.fontDescriptor.withDesign(.rounded) ?? base.fontDescriptor
        UINavigationBar.appearance().largeTitleTextAttributes = [
            .font: UIFontMetrics(forTextStyle: .largeTitle).scaledFont(for: UIFont(descriptor: descriptor, size: 34)),
            .foregroundColor: UIColor(ink)
        ]
    }

    private static func adaptive(_ light: UInt32, _ dark: UInt32) -> Color {
        Color(UIColor { traits in
            let value = traits.userInterfaceStyle == .dark ? dark : light
            return UIColor(red: CGFloat((value >> 16) & 255) / 255,
                           green: CGFloat((value >> 8) & 255) / 255,
                           blue: CGFloat(value & 255) / 255, alpha: 1)
        })
    }

    static let background = adaptive(0xf7f1e7, 0x201f1a)
    static let surface = adaptive(0xfffcf6, 0x2d3028)
    static let ink = adaptive(0x343c31, 0xf7f0e2)
    static let secondary = adaptive(0x646357, 0xc3beaf)
    static let accent = adaptive(0x3c654c, 0xbad0ab)
    static let onAccent = adaptive(0xffffff, 0x201f1a)
    static let border = adaptive(0xe6dfd2, 0x494a3e)
    static let sage = adaptive(0xe6eddd, 0x3e4a37)
    static let clay = adaptive(0x8b503c, 0xe0baa0)
    static let iconBackground = adaptive(0xf3e5d4, 0x473c2d)
    static let departure = adaptive(0xf0e4d2, 0x3c3428)
    static let destructive = adaptive(0x9d3c34, 0xf0b6a5)
}

struct AppAppearance: ViewModifier {
    func body(content: Content) -> some View {
        content
            .tint(AppTheme.accent)
            .foregroundStyle(AppTheme.ink)
            .background(AppTheme.background)
    }
}

struct AppListStyle: ViewModifier {
    func body(content: Content) -> some View {
        content
            .scrollContentBackground(.hidden)
            .background(AppTheme.background)
            .listSectionSpacing(24)
            .environment(\.defaultMinListRowHeight, 52)
    }
}

struct AppPrimaryButtonStyle: ButtonStyle {
    @Environment(\.isEnabled) private var isEnabled
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.headline)
            .frame(maxWidth: .infinity, minHeight: 52)
            .foregroundStyle(isEnabled ? AppTheme.onAccent : AppTheme.secondary)
            .background(isEnabled ? AppTheme.accent : AppTheme.border, in: Capsule())
            .opacity(configuration.isPressed ? 0.75 : 1)
    }
}

struct AppSecondaryButtonStyle: ButtonStyle {
    @Environment(\.isEnabled) private var isEnabled

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.body.weight(.semibold))
            .foregroundStyle(AppTheme.accent)
            .frame(maxWidth: .infinity, minHeight: 52)
            .padding(.horizontal, 16)
            .background(AppTheme.sage, in: Capsule())
            .opacity(isEnabled ? (configuration.isPressed ? 0.7 : 1) : 0.45)
    }
}

struct AppCommitButtonStyle: ButtonStyle {
    @Environment(\.isEnabled) private var isEnabled
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.body.weight(.semibold))
            .padding(.horizontal, 16)
            .frame(minHeight: 44)
            .foregroundStyle(isEnabled ? AppTheme.onAccent : AppTheme.secondary)
            .background(isEnabled ? AppTheme.accent : AppTheme.border, in: Capsule())
            .opacity(configuration.isPressed ? 0.75 : 1)
    }
}

struct WarmIcon: View {
    let symbol: String
    var body: some View {
        Image(systemName: symbol)
            .font(.title3)
            .foregroundStyle(AppTheme.clay)
            .frame(width: 42, height: 42)
            .background(AppTheme.iconBackground, in: RoundedRectangle(cornerRadius: 14))
            .accessibilityHidden(true)
    }
}

extension Color {
    static let appGreen = AppTheme.accent
    static let appRed = AppTheme.destructive
    static let appOrange = AppTheme.clay
}

struct GroupedCardModifier: ViewModifier {
    var radius: CGFloat = 28
    @Environment(\.colorSchemeContrast) private var contrast

    func body(content: Content) -> some View {
        content
            .background(AppTheme.surface, in: RoundedRectangle(cornerRadius: radius))
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
    func appListStyle() -> some View { modifier(AppListStyle()) }
    func appAppearance() -> some View { modifier(AppAppearance()) }
    func groupedBackground() -> some View {
        background(AppTheme.background)
    }

    func groupedCard(radius: CGFloat = 28) -> some View {
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
