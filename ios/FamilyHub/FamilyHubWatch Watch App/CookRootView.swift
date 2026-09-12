import FamilyHubKit
import SwiftUI

/// Routes on the one question that matters: is there a recipe to cook?
///
/// Cook mode is strictly phone-initiated in v1, so there is no browser here and
/// no network call — the whole recipe arrives over WatchConnectivity and the
/// watch cooks offline.
struct CookRootView: View {
    @Environment(WatchSessionStore.self) private var session

    var body: some View {
        NavigationStack {
            if let handoff = session.handoff {
                if (handoff.recipe.steps ?? []).isEmpty {
                    UnavailableView(
                        systemImage: "text.badge.xmark",
                        title: "No steps",
                        message: "\(handoff.recipe.title) has no method to follow."
                    )
                } else {
                    CookSessionView(recipe: handoff.recipe)
                }
            } else {
                UnavailableView(
                    systemImage: "iphone",
                    title: "Nothing to cook",
                    message: "Open a recipe on your iPhone and choose Cook on Watch.",
                    // Nothing arriving is indistinguishable from nothing being
                    // sent, so say which. Only on this screen, and only when
                    // there is genuinely nothing else to look at.
                    footnote: session.diagnostics.summary
                )
            }
        }
    }
}

/// Small stand-in rather than `ContentUnavailableView`, which is cramped at
/// watch sizes.
struct UnavailableView: View {
    let systemImage: String
    let title: String
    let message: String
    var footnote: String?

    var body: some View {
        // Scrollable rather than a bare stack: the diagnostic footnote grows
        // when something has gone wrong, which is exactly when it must stay
        // readable on a 40mm screen.
        ScrollView {
            VStack(spacing: 8) {
                Image(systemName: systemImage)
                    .font(.title2)
                    .foregroundStyle(.secondary)
                Text(title)
                    .font(.headline)
                Text(message)
                    .font(.footnote)
                    .foregroundStyle(.secondary)
                    .multilineTextAlignment(.center)

                if let footnote {
                    Text(footnote)
                        .font(.system(size: 11))
                        .foregroundStyle(.tertiary)
                        .multilineTextAlignment(.center)
                        .padding(.top, 4)
                }
            }
            .frame(maxWidth: .infinity)
            .padding(.horizontal, 8)
        }
    }
}
