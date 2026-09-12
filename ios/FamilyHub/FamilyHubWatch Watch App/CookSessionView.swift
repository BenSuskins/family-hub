import Combine
import FamilyHubKit
import SwiftUI
import WatchKit

/// The cooking carousel: ingredients on the first page, then one step per page.
///
/// Vertical paging rather than the iPhone's horizontal swipe, because vertical
/// pages are driven by the Digital Crown — which is the whole point of cooking
/// from a watch. You can advance with a knuckle without touching the screen.
struct CookSessionView: View {
    let recipe: Recipe

    @Environment(WatchSessionStore.self) private var session
    @Environment(CookTimerScheduler.self) private var timers
    @State private var state: CookState

    init(recipe: Recipe) {
        self.recipe = recipe
        _state = State(initialValue: CookState(recipe: recipe))
    }

    var body: some View {
        TabView(selection: pageBinding) {
            IngredientChecklistView(state: $state)
                .tag(0)

            ForEach(Array(state.steps.enumerated()), id: \.offset) { index, step in
                StepPageView(
                    index: index,
                    step: step,
                    total: state.steps.count,
                    duration: state.duration(forStep: index),
                    onFinish: finish
                )
                .tag(index + 1)
            }
        }
        .tabViewStyle(.verticalPage)
        .navigationTitle(recipe.title)
        .navigationBarTitleDisplayMode(.inline)
        // A page change is a deliberate act; confirm it in the wrist.
        .sensoryFeedback(.selection, trigger: state.currentPage)
    }

    private var pageBinding: Binding<Int> {
        Binding(
            get: { state.currentPage },
            set: { state.go(to: $0) }
        )
    }

    private func finish() {
        Task {
            await timers.cancel()
            session.clearHandoff()
        }
    }
}

/// One step, with its timer if the server recognised a duration in the text.
struct StepPageView: View {
    let index: Int
    let step: String
    let total: Int
    let duration: Int?
    let onFinish: () -> Void

    var body: some View {
        ScrollView {
            VStack(spacing: 10) {
                Text("Step \(index + 1) of \(total)")
                    .font(.caption2)
                    .foregroundStyle(.secondary)

                Text(step)
                    .font(.body)
                    .multilineTextAlignment(.center)

                if let duration {
                    TimerControl(stepIndex: index, duration: duration)
                }

                if index == total - 1 {
                    Button("Done", action: onFinish)
                        .buttonStyle(.borderedProminent)
                        .padding(.top, 4)
                }
            }
            .padding(.horizontal, 4)
        }
    }
}

/// Start/stop for a step's timer, plus a live countdown.
///
/// The countdown is recomputed from the fire time on every tick rather than
/// decremented, so it stays right across the app being suspended — which on
/// watchOS happens constantly.
struct TimerControl: View {
    let stepIndex: Int
    let duration: Int

    @Environment(CookTimerScheduler.self) private var timers
    @State private var now = Date()

    private let tick = Timer.publish(every: 1, on: .main, in: .common).autoconnect()

    var body: some View {
        Group {
            if let active = timers.active, active.stepIndex == stepIndex {
                VStack(spacing: 4) {
                    Text(formatCountdown(active.remaining(at: now)))
                        .font(.system(.title2, design: .rounded).monospacedDigit())
                        .foregroundStyle(.tint)

                    Button("Stop") {
                        Task { await timers.cancel() }
                    }
                    .buttonStyle(.bordered)
                    .controlSize(.mini)
                }
            } else {
                Button {
                    Task { await timers.start(stepIndex: stepIndex, duration: duration, stepText: step) }
                } label: {
                    Label(describeDuration(duration), systemImage: "timer")
                }
                .buttonStyle(.bordered)
                .controlSize(.small)
            }
        }
        .onReceive(tick) { date in
            now = date
            timers.clearIfFired(now: date)
        }
    }

    // Shown in the notification body so the alert says what you were waiting for.
    private var step: String { "Step \(stepIndex + 1)" }
}

/// The ingredient list, tickable so you can keep your place while measuring.
///
/// Keyed by position rather than by text: the iPhone's cook mode keys on
/// "\(group.name)-\(item)", which ticks both boxes when a group repeats an
/// ingredient.
struct IngredientChecklistView: View {
    @Binding var state: CookState

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 6) {
                Text("Ingredients")
                    .font(.headline)

                if state.totalIngredientCount == 0 {
                    Text("None listed")
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                } else {
                    ForEach(Array(state.ingredientGroups.enumerated()), id: \.offset) { groupIndex, group in
                        if !group.name.isEmpty {
                            Text(group.name)
                                .font(.caption2)
                                .foregroundStyle(.secondary)
                                .padding(.top, 4)
                        }

                        ForEach(Array(group.items.enumerated()), id: \.offset) { itemIndex, item in
                            ingredientRow(
                                item: item,
                                index: IngredientIndex(group: groupIndex, item: itemIndex)
                            )
                        }
                    }
                }

                Text("Scroll or turn the crown to start")
                    .font(.caption2)
                    .foregroundStyle(.secondary)
                    .padding(.top, 8)
            }
        }
    }

    private func ingredientRow(item: String, index: IngredientIndex) -> some View {
        let checked = state.isChecked(index)
        return Button {
            state.toggle(index)
        } label: {
            HStack(alignment: .firstTextBaseline, spacing: 6) {
                Image(systemName: checked ? "checkmark.circle.fill" : "circle")
                    .foregroundStyle(checked ? Color.accentColor : Color.secondary)
                Text(item)
                    .font(.footnote)
                    .strikethrough(checked)
                    .foregroundStyle(checked ? .secondary : .primary)
                    .multilineTextAlignment(.leading)
                Spacer(minLength: 0)
            }
        }
        .buttonStyle(.plain)
        .sensoryFeedback(.selection, trigger: checked)
    }
}
