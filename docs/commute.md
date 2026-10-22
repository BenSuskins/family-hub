# Commute times on iPhone

Open Settings from your profile photo on Today. Select Commute.
Enable To work or To home. Select the days and display times for each direction.

Both directions start disabled. To work defaults to Monday–Friday, 06:30–09:30.
To home defaults to Monday–Friday, 16:00–19:30.
The routes are Kelvedon to London Liverpool Street and the reverse.

The schedule uses Europe/London, even when the device uses another time zone.
The start time is inclusive. The end time is exclusive.
An overnight range belongs to its start day and ends the next day.
Select at least one day. Use different start and end times.
Valid changes save immediately. Invalid changes retain the previous valid setting.

The setting stays on this device, separately for each user and server.
It does not sync to other devices. Sign-out retains the setting for that account.

Today shows the next three trains after the meal plan during each active range.
If both directions are active, Today shows To work first.
The card refreshes every 30 seconds while Today is visible and the app is active.
Pull down to refresh manually. Train requests stop when Today is hidden.
Expired train times show an unavailable state. Select Try again to retry.

A changed time shows the scheduled time with a strikethrough, then the expected time.
Departure and arrival estimates can change separately. Unknown estimates keep the scheduled time.
Cancelled trains remain visible. Unknown platforms show “Platform unconfirmed.”
The duration covers train travel, not travel to or from the station.
The API supplies clock times without per-train dates or offsets.
Duration across a clock change can be ambiguous.

The provider returns departures in the next two hours. This feature does not supply an all-day timetable.
It does not send notifications or add a widget.

## Tests

Run the shared package tests with `swift test --package-path ios/FamilyHubKit`.
The app tests cover request cancellation, account changes, schedule boundaries, expiry, and retry.
`CommuteUITests` covers the card, settings, large text, and the profile button location.

To enable the live API test, set `FAMILY_HUB_COMMUTE_TEST_URL` and `FAMILY_HUB_COMMUTE_TEST_TOKEN`.
The test reads both train directions. It does not change server data.
