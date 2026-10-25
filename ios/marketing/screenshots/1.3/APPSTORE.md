# App Store copy

Use this guide to prepare the App Store description and version notes for each
iOS release. The copy-ready files in this folder contain only the text for
their matching App Store Connect fields.

## Release source

Before writing either field, compare the latest iOS release tag with the latest
commit on `main`. Generate both fields from that same release range.

1. Fetch the current `main` branch and tags with `git fetch origin main --tags`.
2. Find the newest iOS release tag with
   `git tag --list 'iOS-*' --sort=-version:refname`. Confirm that it is a shipped
   release tag and check its commit date.
3. Record the tag commit and the latest `origin/main` commit with
   `git show -s --format='%H %cs %s' <tag>` and
   `git show -s --format='%H %cs %s' origin/main`.
4. Review the first-parent changes between them with
   `git log --first-parent --oneline <tag>..origin/main -- ios/FamilyHub ios/FamilyHubKit`.
5. Inspect the relevant diffs and current screens. Commit titles alone do not
   prove that a feature is user-facing or available in the app.

Use `origin/main` as the comparison point so the copy reflects the latest
published main commit, not an older local checkout or the copy-writing branch.
Include user-facing iOS changes in the range. Leave out server maintenance,
dependency updates, CI changes, and marketing asset plumbing unless they
change what an app user can do.

## Description

Write an evergreen explanation of what Family Hub helps a household do. Use the
current app and the marketing screenshots in this version folder to choose the
order and wording of features.

- Lead with shared household planning and a concrete daily benefit.
- Explain the main areas in a useful order: Today, calendar, meals and recipes,
  chores, shopping and inventory, then supporting features such as commute,
  widgets, Siri, Shortcuts, and Apple Watch when they are available.
- State clearly that the app needs a Family Hub server and that the household
  must provide the server address and an account.
- Use short paragraphs and plain-text section headings. Avoid Markdown syntax,
  tables, HTML, keyword stuffing, and claims the current app does not support.
- Keep this field stable across releases. Put release-specific changes in
  What's New instead.

Apple allows up to 4,000 characters for the description. The field accepts
plain text with line breaks; HTML is not supported. See [Apple's platform
version information](https://developer.apple.com/help/app-store-connect/reference/app-information/platform-version-information).

## What's New

Write version notes only about user-facing changes found between the latest
iOS release tag and `origin/main`.

- Put the largest user benefit first.
- Group related changes into up to three short lines.
- Name specific improvements or capabilities. Avoid general claims such as
  “performance improvements” unless the user impact is clear.
- Do not repeat the evergreen description or include internal implementation
  details.
- Keep the text plain and easy to paste. Do not include a title, version number,
  Markdown heading, or surrounding explanation in the copy-ready file.

Apple allows up to 4,000 characters for What's New. This field is required for
updates after the first app version. Use the same Apple reference linked above.

## Files and final review

- `APP-STORE-DESCRIPTION.txt` contains only the Description field text.
- `CHANGELOG.txt` contains only the What's New field text.
- `APPSTORE.md` is the writing guide. Do not paste it into App Store Connect.
- Keep the app name, subtitle, and promotional text as separate App Store
  Connect fields; this workflow does not generate them.
- Check both text files against the current app, the screenshots, and the
  release range. Check each character count before pasting.

For version 1.3, the source range is `iOS-1.2` through `origin/main`. Recheck
both endpoints whenever preparing a later release.
