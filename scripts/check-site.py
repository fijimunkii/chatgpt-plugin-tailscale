#!/usr/bin/env python3
"""Validate the dependency-free validation site with only Python stdlib."""

from __future__ import annotations

import sys
from html.parser import HTMLParser
from pathlib import Path
from urllib.parse import unquote, urlsplit

ROOT = Path(__file__).resolve().parents[1]
SITE = ROOT / "site"


class PageParser(HTMLParser):
    def __init__(self) -> None:
        super().__init__(convert_charrefs=True)
        self.links: list[str] = []
        self.ids: set[str] = set()
        self.has_title = False
        self.has_description = False
        self._in_title = False

    def handle_starttag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        values = {key: value for key, value in attrs}
        if tag == "title":
            self._in_title = True
        if value := values.get("id"):
            self.ids.add(value)
        if tag == "a" and (href := values.get("href")):
            self.links.append(href)
        if (
            tag == "meta"
            and values.get("name") == "description"
            and values.get("content")
        ):
            self.has_description = True

    def handle_startendtag(self, tag: str, attrs: list[tuple[str, str | None]]) -> None:
        self.handle_starttag(tag, attrs)

    def handle_data(self, data: str) -> None:
        if self._in_title and data.strip():
            self.has_title = True

    def handle_endtag(self, tag: str) -> None:
        if tag == "title":
            self._in_title = False


def parse(path: Path) -> PageParser:
    parser = PageParser()
    parser.feed(path.read_text(encoding="utf-8"))
    parser.close()
    return parser


def main() -> int:
    pages = sorted(SITE.glob("*.html"))
    if not pages:
        print("site check: no HTML pages found", file=sys.stderr)
        return 1

    parsed = {page.resolve(): parse(page) for page in pages}
    errors: list[str] = []

    for page, parser in list(parsed.items()):
        rel = page.relative_to(ROOT)
        if not parser.has_title:
            errors.append(f"{rel}: missing non-empty <title>")
        if not parser.has_description:
            errors.append(f"{rel}: missing meta description")

        for raw_href in parser.links:
            href = raw_href.strip()
            if not href or href.startswith(("http://", "https://", "mailto:", "tel:")):
                continue

            parts = urlsplit(href)
            if parts.scheme or parts.netloc:
                continue

            target = (
                (page.parent / unquote(parts.path)).resolve()
                if parts.path
                else page
            )

            try:
                target.relative_to(ROOT)
            except ValueError:
                errors.append(f"{rel}: local link escapes repository: {href}")
                continue

            if not target.exists():
                errors.append(
                    f"{rel}: missing local target for {href}: "
                    f"{target.relative_to(ROOT)}"
                )
                continue

            if parts.fragment and target.suffix.lower() == ".html":
                target_parser = parsed.get(target)
                if target_parser is None:
                    target_parser = parse(target)
                    parsed[target] = target_parser
                fragment = unquote(parts.fragment)
                if fragment not in target_parser.ids:
                    errors.append(
                        f"{rel}: missing fragment #{fragment} in "
                        f"{target.relative_to(ROOT)}"
                    )

    if errors:
        print("site check failed:", file=sys.stderr)
        for error in errors:
            print(f"  - {error}", file=sys.stderr)
        return 1

    print(f"site check: {len(pages)} HTML pages, local links and fragments OK")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
