import SwiftUI

struct CookModeView: View {
    let recipe: Recipe
    @State private var currentPage = 0
    @State private var checkedIngredients: Set<String> = []
    @Environment(\.dismiss) private var dismiss

    private var stepCount: Int { recipe.steps?.count ?? 0 }
    private var totalPages: Int { 1 + stepCount }

    var body: some View {
        GeometryReader { geometry in
            ScrollView(.horizontal) {
                LazyHStack(spacing: 0) {
                    ingredientsPage
                        .frame(width: geometry.size.width, height: geometry.size.height)
                        .id(0)

                    if let steps = recipe.steps {
                        ForEach(Array(steps.enumerated()), id: \.offset) { index, step in
                            stepPage(index: index, step: step, total: steps.count)
                                .frame(width: geometry.size.width, height: geometry.size.height)
                                .id(index + 1)
                        }
                    }
                }
                .scrollTargetLayout()
            }
            .scrollTargetBehavior(.paging)
            .scrollIndicators(.hidden)
        }
        .background(Color(.systemBackground))
        .overlay(alignment: .topTrailing) {
            Button {
                dismiss()
            } label: {
                Image(systemName: "xmark.circle.fill")
                    .font(.title2)
                    .symbolRenderingMode(.hierarchical)
                    .foregroundStyle(.secondary)
            }
            .padding()
        }
        .overlay(alignment: .top) {
            Text(recipe.title)
                .font(.caption.weight(.semibold))
                .foregroundStyle(.secondary)
                .padding(.top, 16)
        }
        .onAppear {
            UIApplication.shared.isIdleTimerDisabled = true
        }
        .onDisappear {
            UIApplication.shared.isIdleTimerDisabled = false
        }
        .statusBarHidden()
    }

    // MARK: - Ingredients Page

    private var ingredientsPage: some View {
        ScrollView(.vertical) {
            VStack(alignment: .leading, spacing: 0) {
                Spacer().frame(height: 60)

                Text("Ingredients")
                    .font(.largeTitle.bold())
                    .padding(.horizontal, 24)
                    .padding(.bottom, 4)

                if let ingredients = recipe.ingredients {
                    ForEach(ingredients, id: \.name) { group in
                        if !group.name.isEmpty {
                            Text(group.name)
                                .font(.title3.weight(.semibold))
                                .foregroundStyle(.secondary)
                                .padding(.horizontal, 24)
                                .padding(.top, 16)
                                .padding(.bottom, 4)
                        }

                        ForEach(group.items, id: \.self) { item in
                            let itemKey = "\(group.name)-\(item)"
                            Button {
                                if checkedIngredients.contains(itemKey) {
                                    checkedIngredients.remove(itemKey)
                                } else {
                                    checkedIngredients.insert(itemKey)
                                }
                            } label: {
                                HStack(spacing: 14) {
                                    Image(systemName: checkedIngredients.contains(itemKey) ? "checkmark.circle.fill" : "circle")
                                        .font(.title3)
                                        .foregroundStyle(checkedIngredients.contains(itemKey) ? Color.accentColor : Color.secondary)

                                    Text(item)
                                        .font(.title3)
                                        .foregroundStyle(checkedIngredients.contains(itemKey) ? .secondary : .primary)
                                        .strikethrough(checkedIngredients.contains(itemKey))
                                }
                                .padding(.horizontal, 24)
                                .padding(.vertical, 8)
                                .frame(maxWidth: .infinity, alignment: .leading)
                                .contentShape(Rectangle())
                            }
                            .buttonStyle(.plain)
                        }
                    }
                } else {
                    Text("No ingredients listed")
                        .font(.title3)
                        .foregroundStyle(.secondary)
                        .padding(.horizontal, 24)
                        .padding(.top, 16)
                }

                Spacer().frame(height: 40)

                HStack {
                    Spacer()
                    Label("Swipe to start cooking", systemImage: "arrow.right")
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                    Spacer()
                }

                Spacer().frame(height: 60)
            }
        }
    }

    // MARK: - Step Page

    private func stepPage(index: Int, step: String, total: Int) -> some View {
        VStack(spacing: 0) {
            Spacer()

            VStack(spacing: 20) {
                Text("Step \(index + 1)")
                    .font(.caption.weight(.bold))
                    .foregroundStyle(.secondary)
                    .textCase(.uppercase)
                    .tracking(1.5)

                Text(step)
                    .font(.title2)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal, 32)

                Text("\(index + 1) of \(total)")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
            }

            Spacer()

            if index == total - 1 {
                Button {
                    dismiss()
                } label: {
                    Text("Done")
                        .font(.headline)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                }
                .buttonStyle(.borderedProminent)
                .padding(.horizontal, 32)
                .padding(.bottom, 40)
            }
        }
    }
}
