#!/usr/bin/env python3
"""Extract iPad screenshots from an XCTest result bundle into a review gallery."""

from __future__ import annotations

import argparse
import html
import json
import re
import shutil
import subprocess
import tempfile
from datetime import datetime
from pathlib import Path


def slug(value: str) -> str:
    value = Path(value).stem
    value = re.sub(r"[^a-zA-Z0-9]+", "-", value).strip("-").lower()
    return value or "screenshot"


def attachment_records(manifest_path: Path) -> list[dict]:
    manifest = json.loads(manifest_path.read_text())
    return manifest if isinstance(manifest, list) else manifest.get("attachments", [])


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("xcresult", type=Path, help="Path to the XCTest .xcresult bundle")
    parser.add_argument(
        "--output-root",
        type=Path,
        default=Path("docs/design-audit/ipad"),
        help="Directory where the indexed screenshot set will be created",
    )
    args = parser.parse_args()

    result = args.xcresult.expanduser().resolve()
    if not result.is_dir():
        parser.error(f"XCTest result bundle does not exist: {result}")

    timestamp = datetime.now().astimezone().strftime("%Y%m%d-%H%M%S")
    destination = args.output_root / f"{result.stem}-{timestamp}"
    captures = destination / "captures"
    destination.mkdir(parents=True, exist_ok=True)
    captures.mkdir(exist_ok=True)

    with tempfile.TemporaryDirectory(prefix="family-hub-ipad-attachments-") as temporary:
        export_directory = Path(temporary)
        subprocess.run(
            [
                "xcrun",
                "xcresulttool",
                "export",
                "attachments",
                "--path",
                str(result),
                "--output-path",
                str(export_directory),
            ],
            check=True,
        )

        records = attachment_records(export_directory / "manifest.json")
        exported: list[tuple[str, str, str, str]] = []
        for test_record in records:
            test_identifier = test_record.get("testIdentifier", "")
            for attachment in test_record.get("attachments", []):
                title = attachment.get("suggestedHumanReadableName", "")
                device = attachment.get("deviceName", "")
                if "ipad" not in device.lower() or not title.lower().endswith(("-ipad-portrait", "-ipad-landscape")):
                    continue

                source = export_directory / attachment["exportedFileName"]
                if not source.is_file() or source.suffix.lower() != ".png":
                    continue

                filename_base = f"{slug(title)}-{slug(test_identifier)}"
                filename = f"{filename_base}.png"
                suffix = 2
                while (captures / filename).exists():
                    filename = f"{filename_base}-{suffix}.png"
                    suffix += 1
                shutil.copy2(source, captures / filename)
                exported.append((title, filename, device, test_identifier))

    if not exported:
        parser.error("No named iPad screenshots were found in this result bundle.")

    cards = []
    for title, filename, device, test_identifier in exported:
        cards.append(
            "<article>"
            f"<img loading=\"lazy\" src=\"captures/{html.escape(filename)}\" alt=\"{html.escape(title)}\">"
            f"<h2>{html.escape(title)}</h2>"
            f"<p>{html.escape(device)} · {html.escape(test_identifier)}</p>"
            "</article>"
        )

    page = """<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Family Hub iPad audit</title>
  <style>
    :root { color-scheme: light dark; font: 16px/1.45 -apple-system, BlinkMacSystemFont, sans-serif; }
    body { margin: 0 auto; max-width: 1600px; padding: 24px; }
    header { margin-bottom: 24px; }
    .screens { display: grid; grid-template-columns: repeat(auto-fit, minmax(min(100%, 360px), 1fr)); gap: 20px; }
    article { min-width: 0; }
    img { display: block; width: 100%; height: auto; border: 1px solid #8886; border-radius: 12px; }
    h2 { font-size: 1rem; margin: 10px 0 2px; }
    p { color: #777; font-size: .82rem; overflow-wrap: anywhere; margin: 0; }
  </style>
</head>
<body>
  <header><h1>Family Hub iPad screenshots</h1><p>Exported XCTest screenshots. Review captures by orientation and screen.</p></header>
  <main class="screens">
""" + "\n".join(cards) + """
  </main>
</body>
</html>
"""
    (destination / "index.html").write_text(page)
    print(f"Wrote {len(exported)} screenshots to {destination}")
    print(f"Open {destination / 'index.html'} to review the set.")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
