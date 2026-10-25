#!/usr/bin/env python3
from html import escape
from html.parser import HTMLParser
from pathlib import Path
import sys


class SlideCopyParser(HTMLParser):
    targets = {"eyebrow", "h1", "dek", "benefits"}
    void_elements = {"area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr"}

    def __init__(self):
        super().__init__(convert_charrefs=True)
        self.captures = {}
        self.current = None
        self.depth = 0

    def handle_starttag(self, tag, attributes):
        attribute_values = dict(attributes)
        target = "h1" if tag == "h1" else next(
            (name for name in self.targets if name in attribute_values.get("class", "").split()),
            None,
        )
        if self.current is None and target is not None:
            self.current = target
            self.captures[target] = []
            self.depth = 1
            return

        if self.current is None:
            return

        rendered_attributes = "".join(
            f' {name}="{escape(value or "", quote=True)}"'
            for name, value in attributes
        )
        self.captures[self.current].append(f"<{tag}{rendered_attributes}>")
        if tag not in self.void_elements:
            self.depth += 1

    def handle_startendtag(self, tag, attributes):
        if self.current is None:
            return
        rendered_attributes = "".join(
            f' {name}="{escape(value or "", quote=True)}"'
            for name, value in attributes
        )
        self.captures[self.current].append(f"<{tag}{rendered_attributes} />")

    def handle_endtag(self, tag):
        if self.current is None or tag in self.void_elements:
            return
        self.depth -= 1
        if self.depth == 0:
            self.current = None
            return
        self.captures[self.current].append(f"</{tag}>")

    def handle_data(self, data):
        if self.current is not None:
            self.captures[self.current].append(escape(data))


def main():
    if len(sys.argv) != 6:
        raise SystemExit("Usage: render-ipad-slide.py source-slide template output capture frame")

    source_slide, template_path, output_path, capture_path, frame_path = map(Path, sys.argv[1:])
    parser = SlideCopyParser()
    parser.feed(source_slide.read_text())
    required_copy = {"eyebrow", "h1", "dek", "benefits"}
    missing = required_copy - parser.captures.keys()
    if missing:
        raise SystemExit(f"Missing marketing copy in {source_slide}: {', '.join(sorted(missing))}")

    replacements = {
        "__EYEBROW__": "".join(parser.captures["eyebrow"]),
        "__HEADLINE__": "".join(parser.captures["h1"]),
        "__DEK__": "".join(parser.captures["dek"]),
        "__BENEFITS__": "".join(parser.captures["benefits"]),
        "__CAPTURE_URL__": capture_path.resolve().as_uri(),
        "__FRAME_URL__": frame_path.resolve().as_uri(),
    }
    rendered = template_path.read_text()
    for marker, value in replacements.items():
        if marker not in rendered:
            raise SystemExit(f"Template marker {marker} is missing from {template_path}")
        rendered = rendered.replace(marker, value)
    output_path.write_text(rendered)


if __name__ == "__main__":
    main()
