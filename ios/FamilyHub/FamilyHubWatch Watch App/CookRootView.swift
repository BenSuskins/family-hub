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
                    message: "Open a recipe on your iPhone and choose Cook on Watch."
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

    var body: some View {
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
        }
        .padding(.horizontal, 8)
    }
}
