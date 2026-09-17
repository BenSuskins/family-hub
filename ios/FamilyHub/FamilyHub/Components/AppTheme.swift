import SwiftUI
import UIKit
import FamilyHubKit

// MARK: - Design tokens

extension Color {
    static let appGreen = Color(.systemGreen)
    static let appRed = Color(.systemRed)
    static let appOrange = Color(.systemOrange)
}

// MARK: - View modifiers

struct MeshBackgroundModifier: ViewModifier {
    func body(content: Content) -> some View {
        content.background(
            ZStack {
                RadialGradient(
                    colors: [Color(red: 0.43, green: 0.47, blue: 0.66).opacity(0.09), .clear],
                    center: .topLeading, startRadius: 0, endRadius: 400
                )
                RadialGradient(
                    colors: [Color(red: 0.36, green: 0.49, blue: 0.60).opacity(0.08), .clear],
                    center: .bottomTrailing, startRadius: 0, endRadius: 350
                )
                Color(UIColor.systemGroupedBackground)
            }
            .ignoresSafeArea()
        )
    }
}

struct GlassCardModifier: ViewModifier {
    var radius: CGFloat = 16

    func body(content: Content) -> some View {
        content
            .background(Color(UIColor.secondarySystemGroupedBackground), in: RoundedRectangle(cornerRadius: radius))
            .shadow(color: .black.opacity(0.04), radius: 2, x: 0, y: 1)
    }
}

extension View {
    func meshBackground() -> some View {
        modifier(MeshBackgroundModifier())
    }

    func glassCard(radius: CGFloat = 16) -> some View {
        modifier(GlassCardModifier(radius: radius))
    }
}

// MARK: - Color hex init

extension Color {
    /// Builds a colour from one of the server's `RRGGBB` calendar colours.
    ///
    /// The parsing lives in ``HexColor`` in FamilyHubKit so the widget
    /// extension — which cannot import this target — gets the same behaviour,
    /// and so the Linux CI job tests it.
    init?(hex: String) {
        guard let parsed = HexColor(hex: hex) else { return nil }
        self.init(red: parsed.red, green: parsed.green, blue: parsed.blue)
    }
}

// MARK: - Section header

struct SectionHeaderLabel: View {
    let text: String
    var color: Color = .secondary

    var body: some View {
        HStack {
            Text(text.uppercased())
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(color)
                .kerning(0.5)
            Spacer()
        }
        .padding(.horizontal, 20)
        .padding(.top, 24)
        .padding(.bottom, 8)
    }
}
