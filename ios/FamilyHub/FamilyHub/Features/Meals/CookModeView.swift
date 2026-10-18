import SwiftUI
import FamilyHubKit

struct CookModeView: View {
    let recipe: Recipe
    @State private var currentPage = 0
    @State private var checkedIngredients: Set<IngredientIndex> = []
    @Environment(\.dismiss) private var dismiss
    @Environment(\.accessibilityReduceMotion) private var reduceMotion

    private var steps: [String] { recipe.steps ?? [] }

    var body: some View {
        NavigationStack {
            TabView(selection: $currentPage) {
                ingredientsPage.tag(0)
                ForEach(Array(steps.enumerated()), id: \.offset) { index, step in
                    ScrollView {
                        VStack(alignment: .leading, spacing: 24) {
                            Text("Step \(index + 1) of \(steps.count)")
                                .font(.headline).foregroundStyle(.secondary)
                            Text(step).font(.title2)
                                .fixedSize(horizontal: false, vertical: true)
                        }
                        .frame(maxWidth: 680, alignment: .leading)
                        .padding(24)
                        .frame(maxWidth: .infinity)
                    }
                    .tag(index + 1)
                }
            }
            .tabViewStyle(.page(indexDisplayMode: .never))
            .navigationTitle(recipe.title)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Close") { dismiss() } }
            }
            .safeAreaInset(edge: .bottom) {
                HStack {
                    Button("Previous", systemImage: "chevron.left") { changePage(by: -1) }
                        .labelStyle(.iconOnly)
                        .frame(minWidth: 44, minHeight: 44)
                        .disabled(currentPage == 0)
                    Spacer()
                    Text("\(currentPage + 1) of \(steps.count + 1)").font(.caption).foregroundStyle(.secondary)
                    Spacer()
                    if currentPage == steps.count {
                        Button("Done") { dismiss() }
                    } else {
                        Button("Next", systemImage: "chevron.right") { changePage(by: 1) }
                            .labelStyle(.iconOnly)
                            .frame(minWidth: 44, minHeight: 44)
                    }
                }
                .buttonStyle(.bordered)
                .padding()
                .background(Color(.systemBackground))
            }
        }
        .modifier(AccessibleMotionModifier())
        .sensoryFeedback(.selection, trigger: currentPage)
        .onAppear { UIApplication.shared.isIdleTimerDisabled = true }
        .onDisappear { UIApplication.shared.isIdleTimerDisabled = false }
    }

    private var ingredientsPage: some View {
        List {
            if (recipe.ingredients ?? []).isEmpty {
                Text("No ingredients listed").foregroundStyle(.secondary)
            }
            ForEach(Array((recipe.ingredients ?? []).enumerated()), id: \.offset) { groupIndex, group in
                Section(group.name.isEmpty ? "Ingredients" : group.name) {
                    ForEach(Array(group.items.enumerated()), id: \.offset) { itemIndex, item in
                        let index = IngredientIndex(group: groupIndex, item: itemIndex)
                        let checked = checkedIngredients.contains(index)
                        Button {
                            if checked { checkedIngredients.remove(index) }
                            else { checkedIngredients.insert(index) }
                        } label: {
                            Label(item, systemImage: checked ? "checkmark.circle.fill" : "circle")
                                .foregroundStyle(.primary)
                                .strikethrough(checked)
                                .multilineTextAlignment(.leading)
                                .frame(minHeight: 44)
                        }
                        .buttonStyle(.borderless)
                        .accessibilityValue(checked ? "Checked" : "Not checked")
                    }
                }
            }
        }
    }

    private func changePage(by offset: Int) {
        withAnimation(reduceMotion ? nil : .smooth) { currentPage += offset }
    }
}
