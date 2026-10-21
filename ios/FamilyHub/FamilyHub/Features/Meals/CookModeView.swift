import SwiftUI
import FamilyHubKit

struct CookModeView: View {
    let recipe: Recipe
    @State private var currentPage = 0
    @State private var checkedIngredients: Set<IngredientIndex> = []
    @Environment(\.dismiss) private var dismiss
    @Environment(\.accessibilityReduceMotion) private var reduceMotion
    @Environment(\.dynamicTypeSize) private var dynamicTypeSize

    private var steps: [String] { recipe.steps ?? [] }

    var body: some View {
        NavigationStack {
            TabView(selection: $currentPage) {
                ingredientsPage.tag(0).accessibilityHidden(currentPage != 0)
                ForEach(Array(steps.enumerated()), id: \.offset) { index, step in
                    ScrollView {
                        VStack(alignment: .leading, spacing: 24) {
                            Text(recipe.title.uppercased())
                                .font(.caption).foregroundStyle(AppTheme.secondary)
                            Text("Step \(index + 1) of \(steps.count)")
                                .font(.system(.largeTitle, design: .rounded, weight: .bold))
                                .accessibilityAddTraits(.isHeader)
                            ProgressView(value: Double(index + 1), total: Double(steps.count))
                                .tint(AppTheme.accent)
                                .accessibilityLabel("Cooking progress")
                                .accessibilityValue("Step \(index + 1) of \(steps.count)")
                            Text(step).font(.title2)
                                .lineSpacing(6)
                                .fixedSize(horizontal: false, vertical: true)
                            if index + 1 < steps.count {
                                Text("Next: " + steps[index + 1])
                                    .font(.subheadline).foregroundStyle(AppTheme.secondary)
                            } else {
                                Text("This is the last step.")
                                    .font(.subheadline).foregroundStyle(AppTheme.secondary)
                            }
                        }
                        .frame(maxWidth: 680, alignment: .leading)
                        .padding(24)
                        .frame(maxWidth: .infinity)
                    }
                    .tag(index + 1)
                    .accessibilityHidden(currentPage != index + 1)
                }
            }
            .tabViewStyle(.page(indexDisplayMode: .never))
            .background(AppTheme.background)
            .navigationTitle(recipe.title)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) { Button("Close") { dismiss() } }
            }
            .safeAreaInset(edge: .bottom) {
                VStack(spacing: 12) {
                    if currentPage > 0 {
                        let layout = dynamicTypeSize.isAccessibilitySize
                            ? AnyLayout(VStackLayout(alignment: .leading, spacing: 8))
                            : AnyLayout(HStackLayout())
                        layout {
                            Button { setPage(0) } label: {
                                Text("Ingredients")
                                    .frame(maxWidth: dynamicTypeSize.isAccessibilitySize ? .infinity : nil, minHeight: 44, alignment: .leading)
                                    .contentShape(Rectangle())
                            }
                            if !dynamicTypeSize.isAccessibilitySize { Spacer() }
                            Button { changePage(by: -1) } label: {
                                Text("Previous step")
                                    .frame(maxWidth: dynamicTypeSize.isAccessibilitySize ? .infinity : nil, minHeight: 44, alignment: .leading)
                                    .contentShape(Rectangle())
                            }
                        }
                    }
                    if currentPage == steps.count {
                        Button("Finish cooking") { dismiss() }
                            .buttonStyle(AppPrimaryButtonStyle())
                    } else {
                        Button(currentPage == 0 ? "Start step 1" : "Next step") { changePage(by: 1) }
                            .buttonStyle(AppPrimaryButtonStyle())
                    }
                }
                .padding(20)
                .background(AppTheme.background)
            }
        }
        .modifier(AccessibleMotionModifier())
        .sensoryFeedback(.selection, trigger: currentPage)
        .onAppear { UIApplication.shared.isIdleTimerDisabled = true }
        .onDisappear { UIApplication.shared.isIdleTimerDisabled = false }
    }

    private var ingredientsPage: some View {
        List {
            Section {
                Text(recipe.title.uppercased()).font(.caption).foregroundStyle(AppTheme.secondary)
                Text("Before you start").font(.system(.largeTitle, design: .rounded, weight: .bold))
                Text("Check the ingredients, then start cooking.").foregroundStyle(AppTheme.secondary)
            }.listRowBackground(Color.clear).listRowSeparator(.hidden)
            if (recipe.ingredients ?? []).isEmpty {
                Text("No ingredients listed").foregroundStyle(AppTheme.secondary)
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
                                .foregroundStyle(AppTheme.ink)
                                .strikethrough(checked)
                                .multilineTextAlignment(.leading)
                                .frame(minHeight: 44)
                        }
                        .buttonStyle(.borderless)
                        .accessibilityValue(checked ? "Checked" : "Not checked")
                    }
                }.listRowBackground(AppTheme.surface)
            }
            Section {
                Text("Your iPhone stays awake while this screen is open.")
                    .font(.footnote).foregroundStyle(AppTheme.secondary)
            }.listRowBackground(Color.clear)
        }
        .appListStyle()
    }

    private func setPage(_ page: Int) {
        withAnimation(reduceMotion ? nil : .smooth) { currentPage = page }
    }

    private func changePage(by offset: Int) { setPage(currentPage + offset) }
}
