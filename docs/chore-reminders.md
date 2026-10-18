# Chore reminders

The server checks family chores at 10:00 and 16:30 in the household timezone.
An admin can change both times in Settings → Chore Reminders.
Family reminders start disabled. Each iPhone also needs notification permission.

The server includes unfinished chores due today and overdue chores.
It includes chores due later today. It excludes future chores and undated pending chores.
Each enabled iPhone receives the same count. A zero count produces no notification.
Tap the notification to open Chores and load current data.

## Configure Apple push notifications

1. Enable Push Notifications for the app identifier in the Apple Developer account.
2. Create an APNs signing key for the app's team.
3. Download the `.p8` key to a private location outside this repository.
4. Set `APNS_KEY_ID` to the key identifier.
5. Set `APNS_TEAM_ID` to the team identifier.
6. Set `APNS_TOPIC` to `uk.co.suskins.FamilyHub`, or the exact bundle identifier of your build.
7. Set `APNS_KEY_PATH` to the readable key path inside the server process.
8. Restart the server through the normal deployment process.

Configure all four `APNS_*` server values together. Leave all empty to disable push support.
The server checks the key at startup. It does not log the key or device tokens.

For Docker Compose, set `APNS_KEY_FILE` to the absolute host path of the `.p8` key.
The optional overlay mounts the key read-only and sets `APNS_KEY_PATH` inside the container:

```sh
cd server
docker compose -f docker-compose.prod.yml -f docker-compose.push.yml up -d
```

This command changes the local deployment. Run it only when you intend to deploy.
The standard Compose file still works without push credentials or the overlay.

## Enable reminders

1. Install an iPhone build with the Push Notifications entitlement.
2. Open Settings → Chore Reminders as an admin.
3. Check the household timezone and both times.
4. Enable Family Reminders and tap Save.
5. Enable Notifications on This iPhone on each family iPhone.
6. Allow notifications when iOS asks.

Debug builds use the APNs sandbox. Release builds use production APNs.
Both the entitlement and the registration environment use the same build setting.
Use the matching signing profile when you install a build on a physical iPhone.

When notifications are enabled, the iPhone needs a server connection to disable them, sign out, or change servers.
The app removes its registration before it completes those actions.
Sign-out works offline when notifications are disabled.
If Apple permission changes, the app reconciles the registration when it becomes active.

## Delivery and verification

The server checks the schedule every 30 seconds, including on startup.
SQLite records prevent normal duplicate deliveries across restarts.
Transient failures can retry for five minutes. Each retry checks current chores again.
The server skips expired reminders and removes tokens that Apple reports as invalid.
Delivery records expire after seven days.

Delivery is best effort. Focus, Notification Summary, connectivity, and Apple delivery can delay an alert.
A network failure after Apple accepts a request can leave its result unknown.
Retries reuse the delivery identifier and collapse identifier, but exactly-once delivery is not guaranteed.

Verify a signed build on a physical iPhone before release:

- Check an alert with the app closed and an unfinished chore due today.
- Check that the alert opens Chores and loads current data.
- Complete all due chores before the second time and check that no alert appears.
- Check another family iPhone and both Apple push environments used for release.
- Check disablement, sign-out, and a server change.

The automated tests use a fake sender and a local HTTP/2 server.
They do not prove delivery through Apple to a physical iPhone.

## Admin family broadcasts

Open Settings → Family Broadcast as an admin.
Enter a message of up to 500 Unicode code points and tap Send.
The message goes to every family iPhone with Notifications on This iPhone enabled.
Broadcasts work even when Family Reminders is disabled.

The result shows how many notifications Apple accepts and how many fail or remain pending.
Tap Send again to retry eligible failures within five minutes.
The app reuses the request identifier to avoid repeating accepted notifications.
Edit the text to start a new message. The server retains messages for 30 days.
Tap a received family message to open Today.
