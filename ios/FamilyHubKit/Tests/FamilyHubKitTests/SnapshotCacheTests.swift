import XCTest
@testable import FamilyHubKit

final class SnapshotCacheTests: XCTestCase {
    private struct Payload: Codable, Equatable, Sendable {
        let value: String
    }

    func testSnapshotPersistsMetadataAndSeparatesAccounts() async throws {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        defer { try? FileManager.default.removeItem(at: directory) }
        let cache = SnapshotCache(directory: directory)
        let now = Date(timeIntervalSince1970: 1_800_000_000)
        await cache.write(try JSONEncoder().encode(Payload(value: "family one")), namespace: "server-user-one", key: "meals-week", freshness: 60, now: now)
        await cache.write(try JSONEncoder().encode(Payload(value: "family two")), namespace: "server-user-two", key: "meals-week", freshness: 60, now: now)

        let first = await cache.read(namespace: "server-user-one", key: "meals-week")
        let second = await cache.read(namespace: "server-user-two", key: "meals-week")
        XCTAssertEqual(first.flatMap { try? JSONDecoder().decode(Payload.self, from: $0.data) }, Payload(value: "family one"))
        XCTAssertEqual(first?.metadata.expiresAt, now.addingTimeInterval(60))
        XCTAssertEqual(second.flatMap { try? JSONDecoder().decode(Payload.self, from: $0.data) }, Payload(value: "family two"))

        let relaunchedCache = SnapshotCache(directory: directory)
        let afterRelaunch = await relaunchedCache.read(namespace: "server-user-one", key: "meals-week")
        XCTAssertEqual(afterRelaunch.flatMap { try? JSONDecoder().decode(Payload.self, from: $0.data) }, Payload(value: "family one"))
    }

    func testCorruptSnapshotIsIgnored() async throws {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        defer { try? FileManager.default.removeItem(at: directory) }
        let cache = SnapshotCache(directory: directory)
        await cache.write(try JSONEncoder().encode(Payload(value: "valid")), namespace: "server-user", key: "dashboard-today", freshness: 60)
        let files = try FileManager.default.contentsOfDirectory(at: directory, includingPropertiesForKeys: nil)
        try Data("not json".utf8).write(to: files[0], options: .atomic)

        let snapshot = await cache.read(namespace: "server-user", key: "dashboard-today")
        XCTAssertNil(snapshot)
    }

    func testOldSnapshotRemainsReadableAfterFreshnessExpires() async throws {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        defer { try? FileManager.default.removeItem(at: directory) }
        let cache = SnapshotCache(directory: directory)
        await cache.write(try JSONEncoder().encode(Payload(value: "offline data")), namespace: "server-user", key: "calendar-2026-10", freshness: 1, now: Date(timeIntervalSince1970: 1))

        let snapshot = await cache.read(namespace: "server-user", key: "calendar-2026-10")
        XCTAssertEqual(snapshot.flatMap { try? JSONDecoder().decode(Payload.self, from: $0.data) }?.value, "offline data")
        XCTAssertFalse(snapshot?.metadata.isFresh ?? true)
    }

    func testDiskImageCacheKeepsVariantKeysSeparateAndEnforcesItsLimit() async {
        let directory = FileManager.default.temporaryDirectory.appendingPathComponent(UUID().uuidString)
        defer { try? FileManager.default.removeItem(at: directory) }
        let cache = DiskImageCache(directory: directory, maximumBytes: 10)
        await cache.write(Data("thumb".utf8), namespace: "account", key: "recipe-thumbnail")
        await cache.write(Data("large".utf8), namespace: "account", key: "recipe-original")
        await cache.write(Data("too-large!!".utf8), namespace: "account", key: "another")

        let thumbnail = await cache.read(namespace: "account", key: "recipe-thumbnail")
        let original = await cache.read(namespace: "account", key: "recipe-original")
        let oversized = await cache.read(namespace: "account", key: "another")
        XCTAssertEqual(thumbnail, Data("thumb".utf8))
        XCTAssertEqual(original, Data("large".utf8))
        XCTAssertNil(oversized)
    }
}
