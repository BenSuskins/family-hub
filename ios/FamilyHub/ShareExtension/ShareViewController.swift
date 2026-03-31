import UIKit
import SwiftUI
import UniformTypeIdentifiers

final class ShareViewController: UIViewController {

    override func viewDidLoad() {
        super.viewDidLoad()
        Task { await presentShareUI() }
    }

    private func presentShareUI() async {
        let sharedURL = await extractURL()

        let defaults = UserDefaults(suiteName: "group.uk.co.suskins.familyhub")
        let baseURL = defaults?.string(forKey: "baseURL") ?? ""
        let apiToken = defaults?.string(forKey: "api_token") ?? ""

        let content: AnyView
        if baseURL.isEmpty || apiToken.isEmpty {
            content = AnyView(notConfiguredView())
        } else if let url = sharedURL {
            content = AnyView(ShareView(
                sharedURL: url,
                baseURL: baseURL,
                apiToken: apiToken,
                onDismiss: { [weak self] in self?.complete() }
            ))
        } else {
            content = AnyView(noURLView())
        }

        let host = UIHostingController(rootView: content)
        host.modalPresentationStyle = .pageSheet
        addChild(host)
        view.addSubview(host.view)
        host.view.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            host.view.topAnchor.constraint(equalTo: view.topAnchor),
            host.view.bottomAnchor.constraint(equalTo: view.bottomAnchor),
            host.view.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            host.view.trailingAnchor.constraint(equalTo: view.trailingAnchor),
        ])
        host.didMove(toParent: self)
    }

    private func extractURL() async -> URL? {
        guard let items = extensionContext?.inputItems as? [NSExtensionItem] else { return nil }
        for item in items {
            for provider in item.attachments ?? [] {
                if provider.hasItemConformingToTypeIdentifier(UTType.url.identifier) {
                    if let url = try? await provider.loadItem(forTypeIdentifier: UTType.url.identifier) as? URL {
                        return url
                    }
                }
                if provider.hasItemConformingToTypeIdentifier(UTType.plainText.identifier) {
                    if let text = try? await provider.loadItem(forTypeIdentifier: UTType.plainText.identifier) as? String,
                       let url = URL(string: text), url.scheme?.hasPrefix("http") == true {
                        return url
                    }
                }
            }
        }
        return nil
    }

    private func complete() {
        extensionContext?.completeRequest(returningItems: nil)
    }

    private func notConfiguredView() -> some View {
        VStack(spacing: 16) {
            Image(systemName: "exclamationmark.triangle")
                .font(.largeTitle)
                .foregroundStyle(.orange)
            Text("Family Hub Not Set Up")
                .font(.headline)
            Text("Please open Family Hub and sign in before using the share extension.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
            Button("Dismiss") { self.complete() }
                .buttonStyle(.borderedProminent)
        }
        .padding()
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private func noURLView() -> some View {
        VStack(spacing: 16) {
            Image(systemName: "link.badge.plus")
                .font(.largeTitle)
                .foregroundStyle(.secondary)
            Text("No URL Found")
                .font(.headline)
            Text("Share a web page URL to add it as a recipe.")
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .multilineTextAlignment(.center)
            Button("Dismiss") { self.complete() }
                .buttonStyle(.borderedProminent)
        }
        .padding()
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}
