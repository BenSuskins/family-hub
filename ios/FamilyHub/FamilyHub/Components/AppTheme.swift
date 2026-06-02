import SwiftUI
import UIKit

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
    init?(hex: String) {
        let cleaned = hex.trimmingCharacters(in: .whitespacesAndNewlines).replacingOccurrences(of: "#", with: "")
        guard cleaned.count == 6, let rgb = UInt64(cleaned, radix: 16) else { return nil }
        self.init(
            red: Double((rgb >> 16) & 0xFF) / 255,
            green: Double((rgb >> 8) & 0xFF) / 255,
            blue: Double(rgb & 0xFF) / 255
        )
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
