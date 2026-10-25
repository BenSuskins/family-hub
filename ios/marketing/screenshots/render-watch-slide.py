#!/usr/bin/env python3
from html import escape
from pathlib import Path
import sys


SLIDES = {
    "ingredients": {
        "capture": "01-watch-ingredients.png",
        "headline": "Your recipe, on your wrist",
        "dek": "Check off ingredients as you cook.",
        "benefits": "Keep your place · Cook offline",
    },
    "step": {
        "capture": "02-watch-step.png",
        "headline": "Keep cooking, hands-free",
        "dek": "A six-minute timer keeps the chicken on track.",
        "benefits": "Step-by-step cooking · Built-in timers",
    },
}


def main():
    if len(sys.argv) != 5:
        raise SystemExit("Usage: render-watch-slide.py slide template output-directory capture-directory")

    slide_name, template_path, output_directory, capture_directory = sys.argv[1:]
    if slide_name not in SLIDES:
        raise SystemExit(f"Unknown Watch slide: {slide_name}")

    slide = SLIDES[slide_name]
    capture_path = Path(capture_directory, slide["capture"]).resolve()
    if not capture_path.is_file():
        raise SystemExit(f"Missing Watch capture: {capture_path}")

    rendered = Path(template_path).read_text()
    replacements = {
        "__HEADLINE__": escape(slide["headline"]),
        "__DEK__": escape(slide["dek"]),
        "__BENEFITS__": escape(slide["benefits"]),
        "__CAPTURE_URL__": capture_path.as_uri(),
    }
    for marker, value in replacements.items():
        if marker not in rendered:
            raise SystemExit(f"Template marker {marker} is missing from {template_path}")
        rendered = rendered.replace(marker, value)

    output_directory = Path(output_directory)
    output_directory.mkdir(parents=True, exist_ok=True)
    (output_directory / f"watch-{slide_name}.html").write_text(rendered)


if __name__ == "__main__":
    main()
