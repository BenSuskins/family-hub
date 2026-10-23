import Foundation

public struct SnapshotMetadata: Codable, Sendable {
    public let savedAt: Date
    public let expiresAt: Date

    public var isFresh: Bool { Date() < expiresAt }
}

public struct CacheFreshness: Sendable {
    public let dashboard: TimeInterval = 60
    public let meals: TimeInterval = 60
    public let chores: TimeInterval = 60
    public let calendar: TimeInterval = 5 * 60
    public let recipes: TimeInterval = 5 * 60
    public let users: TimeInterval = 15 * 60
    public let categories: TimeInterval = 15 * 60

    public init() {}
}

public struct SnapshotData: Sendable {
    public let data: Data
    public let metadata: SnapshotMetadata
}

/// Versioned disk snapshots keyed by an opaque account namespace and resource key.
public actor SnapshotCache {
    private struct Envelope: Codable {
        let version: Int
        let savedAt: Date
        let expiresAt: Date
        let data: Data
    }

    private let directory: URL
    private let version = 1
    private var generations: [String: Int] = [:]

    public init(directory: URL? = nil) {
        if let directory {
            self.directory = directory
        } else {
            let base = FileManager.default.urls(for: .cachesDirectory, in: .userDomainMask).first
                ?? FileManager.default.temporaryDirectory
            self.directory = base.appendingPathComponent("FamilyHubSnapshots", isDirectory: true)
        }
    }

    public func read(namespace: String, key: String) -> SnapshotData? {
        guard let data = try? Data(contentsOf: fileURL(namespace: namespace, key: key)),
              let envelope = try? JSONDecoder().decode(Envelope.self, from: data),
              envelope.version == version else {
            return nil
        }
        return SnapshotData(
            data: envelope.data,
            metadata: SnapshotMetadata(savedAt: envelope.savedAt, expiresAt: envelope.expiresAt)
        )
    }

    public func write(
        _ data: Data,
        namespace: String,
        key: String,
        freshness: TimeInterval,
        generation: Int = 0,
        now: Date = Date()
    ) {
        guard generations[namespace] == nil || generations[namespace] == generation else { return }
        do {
            try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
            let envelope = Envelope(version: version, savedAt: now, expiresAt: now.addingTimeInterval(freshness), data: data)
            let data = try JSONEncoder().encode(envelope)
            try data.write(to: fileURL(namespace: namespace, key: key), options: .atomic)
        } catch {
            // The network response remains usable when the device cannot persist its cache.
        }
    }

    public func activate(namespace: String, generation: Int) {
        if generation >= (generations[namespace] ?? 0) { generations[namespace] = generation }
    }

    public func remove(namespace: String, generation: Int? = nil) {
        if let generation { generations[namespace] = generation }
        guard let files = try? FileManager.default.contentsOfDirectory(at: directory, includingPropertiesForKeys: nil) else { return }
        for file in files where file.lastPathComponent.hasPrefix(Self.safe(namespace) + "-") {
            try? FileManager.default.removeItem(at: file)
        }
    }

    public func remove(namespace: String, key: String) {
        try? FileManager.default.removeItem(at: fileURL(namespace: namespace, key: key))
    }

    private func fileURL(namespace: String, key: String) -> URL {
        directory.appendingPathComponent("\(Self.safe(namespace))-\(Self.safe(key)).json")
    }

    private static func safe(_ value: String) -> String {
        value.utf8.reduce(into: UInt64(14_695_981_039_346_656_037)) { hash, byte in
            hash = (hash ^ UInt64(byte)) &* 1_099_511_628_211
        }.description
    }
}

public actor DiskImageCache {
    private let directory: URL
    private let maximumBytes: Int
    private var generations: [String: Int] = [:]

    public init(directory: URL? = nil, maximumBytes: Int = 50 * 1024 * 1024) {
        if let directory {
            self.directory = directory
        } else {
            let base = FileManager.default.urls(for: .cachesDirectory, in: .userDomainMask).first
                ?? FileManager.default.temporaryDirectory
            self.directory = base.appendingPathComponent("FamilyHubImages", isDirectory: true)
        }
        self.maximumBytes = max(0, maximumBytes)
    }

    public func read(namespace: String, key: String) -> Data? {
        let url = fileURL(namespace: namespace, key: key)
        guard let data = try? Data(contentsOf: url) else { return nil }
        try? FileManager.default.setAttributes([.modificationDate: Date()], ofItemAtPath: url.path)
        return data
    }

    public func write(_ data: Data, namespace: String, key: String, generation: Int = 0) {
        guard generations[namespace] == nil || generations[namespace] == generation else { return }
        guard data.count <= maximumBytes else { return }
        do {
            try FileManager.default.createDirectory(at: directory, withIntermediateDirectories: true)
            try data.write(to: fileURL(namespace: namespace, key: key), options: .atomic)
            let files = try FileManager.default.contentsOfDirectory(
                at: directory,
                includingPropertiesForKeys: [.fileSizeKey, .contentModificationDateKey]
            )
            let entries = files.compactMap { url -> (URL, Int, Date)? in
                guard let values = try? url.resourceValues(forKeys: [.fileSizeKey, .contentModificationDateKey]),
                      let size = values.fileSize else { return nil }
                return (url, size, values.contentModificationDate ?? .distantPast)
            }.sorted { $0.2 < $1.2 }
            var total = entries.reduce(0) { $0 + $1.1 }
            for (url, size, _) in entries where total > maximumBytes {
                try? FileManager.default.removeItem(at: url)
                total -= size
            }
        } catch {
            // Images are optional. A disk failure must not fail the API request.
        }
    }

    public func activate(namespace: String, generation: Int) {
        if generation >= (generations[namespace] ?? 0) { generations[namespace] = generation }
    }

    public func remove(namespace: String, generation: Int? = nil) {
        if let generation { generations[namespace] = generation }
        guard let files = try? FileManager.default.contentsOfDirectory(at: directory, includingPropertiesForKeys: nil) else { return }
        for file in files where file.lastPathComponent.hasPrefix(Self.safe(namespace) + "-") {
            try? FileManager.default.removeItem(at: file)
        }
    }

    public func remove(namespace: String, key: String) {
        try? FileManager.default.removeItem(at: fileURL(namespace: namespace, key: key))
    }

    private func fileURL(namespace: String, key: String) -> URL {
        directory.appendingPathComponent("\(Self.safe(namespace))-\(Self.safe(key)).image")
    }

    private static func safe(_ value: String) -> String {
        value.utf8.reduce(into: UInt64(14_695_981_039_346_656_037)) { hash, byte in
            hash = (hash ^ UInt64(byte)) &* 1_099_511_628_211
        }.description
    }
}
