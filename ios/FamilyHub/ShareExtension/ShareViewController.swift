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

    /// TikTok and Instagram do not hand over a tidy URL attachment: they share a
    /// blob of text such as "Check out this recipe https://vm.tiktok.com/ZGx…/",
    /// sometimes alongside a preview image. So take a web URL when one is
    /// offered, and otherwise dig a link out of whatever text we were given.
    private func extractURL() async -> URL? {
        guard let items = extensionContext?.inputItems as? [NSExtensionItem] else { return nil }

        var textCandidates: [String] = []

        for item in items {
            for provider in item.attachments ?? [] {
                if let url = await loadWebURL(from: provider) {
                    return url
                }
                if let text = await loadText(from: provider) {
                    textCandidates.append(text)
                }
            }
            if let attributed = item.attributedContentText?.string {
                textCandidates.append(attributed)
            }
            if let titleText = item.attributedTitle?.string {
                textCandidates.append(titleText)
            }
        }

        for text in textCandidates {
            if let url = Self.firstWebURL(in: text) {
                return url
            }
        }
        return nil
    }

    private func loadWebURL(from provider: NSItemProvider) async -> URL? {
        guard provider.hasItemConformingToTypeIdentifier(UTType.url.identifier) else { return nil }
        guard let item = try? await provider.loadItem(forTypeIdentifier: UTType.url.identifier) else { return nil }

        let url: URL?
        switch item {
        case let value as URL: url = value
        case let value as String: url = URL(string: value)
        case let value as Data: url = String(data: value, encoding: .utf8).flatMap(URL.init(string:))
        default: url = nil
        }

        guard let url, let scheme = url.scheme?.lowercased(), scheme == "http" || scheme == "https" else {
            return nil
        }
        return url
    }

    private func loadText(from provider: NSItemProvider) async -> String? {
        for identifier in [UTType.plainText.identifier, UTType.text.identifier, UTType.utf8PlainText.identifier] {
            guard provider.hasItemConformingToTypeIdentifier(identifier) else { continue }
            if let text = try? await provider.loadItem(forTypeIdentifier: identifier) as? String, !text.isEmpty {
                return text
            }
        }
        return nil
    }

    /// Pulls the first http(s) link out of free text, which is how the social
    /// apps pass the post along.
    static func firstWebURL(in text: String) -> URL? {
        let trimmed = text.trimmingCharacters(in: .whitespacesAndNewlines)
        if let url = URL(string: trimmed), let scheme = url.scheme?.lowercased(),
           scheme == "http" || scheme == "https" {
            return url
        }

        guard let detector = try? NSDataDetector(types: NSTextCheckingResult.CheckingType.link.rawValue) else {
            return nil
        }
        let range = NSRange(trimmed.startIndex..<trimmed.endIndex, in: trimmed)
        for match in detector.matches(in: trimmed, options: [], range: range) {
            guard let url = match.url, let scheme = url.scheme?.lowercased(),
                  scheme == "http" || scheme == "https" else { continue }
            return url
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
            Text("No Link Found")
                .font(.headline)
            Text("Share a recipe page, or a TikTok or Instagram post, to add it as a recipe.")
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
