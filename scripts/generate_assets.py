#!/usr/bin/env python3
"""Generate Fabricator's pixel-art brand assets.

Everything the project ships as a logo, banner, favicon, or social image is
drawn here from a pixel grid, so the art is reproducible and reviewable as
source rather than committed as opaque binaries.

    python3 scripts/generate_assets.py

Writes SVGs directly and rasterises the PNGs with `rsvg-convert`.
"""

from __future__ import annotations

import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
MEDIA = ROOT / "docs" / "media"
WEB_ASSETS = ROOT / "website" / "src" / "assets"
WEB_PUBLIC = ROOT / "website" / "public"

# Go's cyan, warmed with a forge amber for the "fabrication" half of the idea.
INK = "#0b1020"
CUBE_TOP = "#7ee8fa"
CUBE_LEFT = "#00add8"
CUBE_RIGHT = "#00728f"
CUBE_EDGE = "#0b1020"
AMBER = "#f2a93b"
AMBER_DIM = "#a86f1d"
WORD = "#f2f5fa"
MUTED = "#8b97a8"
GRID = "#16203a"

# A 5x7 pixel font. Only the glyphs the brand needs are defined; the wordmark
# is the one piece of text that must stay pixel art at every size.
FONT: dict[str, list[str]] = {
    "A": [".###.", "#...#", "#...#", "#####", "#...#", "#...#", "#...#"],
    "B": ["####.", "#...#", "#...#", "####.", "#...#", "#...#", "####."],
    "C": [".####", "#....", "#....", "#....", "#....", "#....", ".####"],
    "D": ["####.", "#...#", "#...#", "#...#", "#...#", "#...#", "####."],
    "E": ["#####", "#....", "#....", "####.", "#....", "#....", "#####"],
    "F": ["#####", "#....", "#....", "####.", "#....", "#....", "#...."],
    "G": [".####", "#....", "#....", "#..##", "#...#", "#...#", ".###."],
    "H": ["#...#", "#...#", "#...#", "#####", "#...#", "#...#", "#...#"],
    "I": ["#####", "..#..", "..#..", "..#..", "..#..", "..#..", "#####"],
    "J": ["....#", "....#", "....#", "....#", "....#", "#...#", ".###."],
    "K": ["#...#", "#..#.", "#.#..", "##...", "#.#..", "#..#.", "#...#"],
    "L": ["#....", "#....", "#....", "#....", "#....", "#....", "#####"],
    "M": ["#...#", "##.##", "#.#.#", "#.#.#", "#...#", "#...#", "#...#"],
    "N": ["#...#", "##..#", "#.#.#", "#.#.#", "#..##", "#...#", "#...#"],
    "O": [".###.", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."],
    "P": ["####.", "#...#", "#...#", "####.", "#....", "#....", "#...."],
    "Q": [".###.", "#...#", "#...#", "#...#", "#.#.#", "#..#.", ".##.#"],
    "R": ["####.", "#...#", "#...#", "####.", "#..#.", "#...#", "#...#"],
    "S": [".####", "#....", "#....", ".###.", "....#", "....#", "####."],
    "T": ["#####", "..#..", "..#..", "..#..", "..#..", "..#..", "..#.."],
    "U": ["#...#", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."],
    "V": ["#...#", "#...#", "#...#", "#...#", "#...#", ".#.#.", "..#.."],
    "W": ["#...#", "#...#", "#...#", "#.#.#", "#.#.#", "##.##", "#...#"],
    "X": ["#...#", "#...#", ".#.#.", "..#..", ".#.#.", "#...#", "#...#"],
    "Y": ["#...#", "#...#", ".#.#.", "..#..", "..#..", "..#..", "..#.."],
    "Z": ["#####", "....#", "...#.", "..#..", ".#...", "#....", "#####"],
}


GLYPH_W, GLYPH_H, TRACKING = 5, 7, 1


class Canvas:
    """A pixel grid that renders to SVG as one <rect> per run of pixels."""

    def __init__(self, width: int, height: int) -> None:
        self.width = width
        self.height = height
        self.pixels: dict[tuple[int, int], str] = {}

    def set(self, x: int, y: int, color: str) -> None:
        if 0 <= x < self.width and 0 <= y < self.height:
            self.pixels[(x, y)] = color

    def rect(self, x: int, y: int, w: int, h: int, color: str) -> None:
        for dy in range(h):
            for dx in range(w):
                self.set(x + dx, y + dy, color)

    def blit(self, grid: list[str], x: int, y: int, palette: dict[str, str], scale: int = 1) -> None:
        for row, line in enumerate(grid):
            for col, char in enumerate(line):
                color = palette.get(char)
                if color is None:
                    continue
                self.rect(x + col * scale, y + row * scale, scale, scale, color)

    def text(self, message: str, x: int, y: int, color: str, scale: int = 1) -> int:
        """Draw message in the pixel font. Returns the width drawn."""
        cursor = x
        for char in message:
            if char == " ":
                cursor += (GLYPH_W + TRACKING) * scale
                continue
            glyph = FONT[char]
            self.blit(glyph, cursor, y, {"#": color}, scale)
            cursor += (GLYPH_W + TRACKING) * scale
        return cursor - x - TRACKING * scale

    def to_svg(self, scale: int, title: str, background: str | None = None, extra: str = "") -> str:
        w, h = self.width * scale, self.height * scale
        parts = [
            "<svg xmlns='http://www.w3.org/2000/svg' "
            f"viewBox='0 0 {w} {h}' width='{w}' height='{h}' "
            f"role='img' aria-label='{title}' shape-rendering='crispEdges'>",
            f"<title>{title}</title>",
        ]
        if background:
            parts.append(f"<rect width='{w}' height='{h}' fill='{background}'/>")
        parts.append(extra)

        # Merge horizontally adjacent pixels of the same colour into one rect so
        # the SVG stays small enough to sit in a README without bloating it.
        for y in range(self.height):
            x = 0
            while x < self.width:
                color = self.pixels.get((x, y))
                if color is None:
                    x += 1
                    continue
                run = 1
                while self.pixels.get((x + run, y)) == color:
                    run += 1
                parts.append(
                    f"<rect x='{x * scale}' y='{y * scale}' width='{run * scale}' height='{scale}' fill='{color}'/>"
                )
                x += run
        parts.append("</svg>")
        return "\n".join(parts) + "\n"


# The two cube sizes, authored explicitly rather than derived. A 2:1 isometric
# silhouette only reads as a cube when every slope lands on exact pixel pairs,
# and a formula that is one row out anywhere turns it into a blob.
CUBE_16 = [
    ".......TT.......",
    ".....TTTTTT.....",
    "...TTTTTTTTTT...",
    ".TTTTTTTTTTTTTT.",
    ".TTTTTTTTTTTTTT.",
    ".LLTTTTTTTTTTRR.",
    ".LLLLTTTTTTRRRR.",
    ".LLLLLLTTRRRRRR.",
    ".LLLLLLLRRRRRRR.",
    ".LLLLLLLRRRRRRR.",
    "...LLLLLRRRRR...",
    ".....LLLRRR.....",
    ".......LR.......",
]


def cube() -> list[str]:
    """The fabricated object: a 16-pixel isometric cube.

    Only one size is authored. Smaller cubes are this artwork at scale 1 and
    larger ones are it scaled up, because a hand-drawn 8-pixel version loses
    the distinction between the top face and the sides and reads as a plus.
    """
    return CUBE_16


CUBE_PALETTE = {"T": CUBE_TOP, "L": CUBE_LEFT, "R": CUBE_RIGHT, "E": CUBE_EDGE}
CUBE_FLAT = {"T": AMBER, "L": AMBER_DIM, "R": AMBER_DIM, "E": CUBE_EDGE}


def draw_grid_backdrop(canvas: Canvas, step: int = 8) -> None:
    for y in range(0, canvas.height, step):
        for x in range(0, canvas.width, step):
            canvas.set(x, y, GRID)


def build_mark() -> Canvas:
    """The square logo: one cube, centred on a 16x16 grid, no background."""
    art = cube()
    canvas = Canvas(16, 16)
    canvas.blit(art, 0, (16 - len(art)) // 2, CUBE_PALETTE)
    return canvas


def _wordmark(canvas: Canvas, x: int, y: int, scale: int = 2) -> int:
    """The name, plus the cyan rule and amber dashes that sit under it."""
    width = canvas.text("FABRICATOR", x, y, WORD, scale=scale)
    rule = y + GLYPH_H * scale + 2
    for column in range(x, x + width):
        canvas.set(column, rule, CUBE_LEFT)
    for column in range(x, x + width, 3):
        canvas.set(column, rule + 6, AMBER)
    return width


def _belt(canvas: Canvas, x: int, y: int, count: int = 3, scale: int = 1) -> None:
    """A short conveyor: the first cube solid, the rest stamped copies."""
    small = cube()
    gap = 16 * scale + 4
    for index in range(count):
        palette = CUBE_PALETTE if index == 0 else CUBE_FLAT
        canvas.blit(small, x + index * gap, y, palette, scale=scale)
    span = (count - 1) * gap + 16 * scale
    for column in range(x - 2, x + span + 2, 2):
        canvas.set(column, y + len(small) * scale + 3, MUTED)


TAGLINE = "TYPED TEST DATA FOR GO"


def build_banner() -> Canvas:
    """The wide README and docs hero: the machine, its output, then the name."""
    canvas = Canvas(280, 64)
    draw_grid_backdrop(canvas)
    canvas.blit(cube(), 14, 18, CUBE_PALETTE, scale=2)
    _belt(canvas, 58, 24)
    _wordmark(canvas, 134, 12)
    canvas.text(TAGLINE, 134, 44, MUTED)
    return canvas


def build_social() -> Canvas:
    """The 1280x640 GitHub social preview, drawn at 160x80 and scaled 8x."""
    canvas = Canvas(160, 80)
    draw_grid_backdrop(canvas)
    canvas.blit(cube(), 64, 6, CUBE_PALETTE, scale=2)
    _wordmark(canvas, 21, 38)
    canvas.text(TAGLINE, 15, 66, MUTED)
    return canvas


def rasterise(svg: Path, png: Path, width: int, height: int) -> None:
    if shutil.which("rsvg-convert") is None:
        print(f"skip {png.name}: rsvg-convert not installed", file=sys.stderr)
        return
    subprocess.run(
        ["rsvg-convert", "-w", str(width), "-h", str(height), str(svg), "-o", str(png)],
        check=True,
    )


def write(path: Path, content: str) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(content)
    print(f"wrote {path.relative_to(ROOT)}")


def main() -> None:
    mark = build_mark()
    mark_svg = mark.to_svg(scale=16, title="Fabricator")
    write(MEDIA / "fabricator-mark.svg", mark_svg)
    write(WEB_ASSETS / "logo.svg", mark_svg)
    write(WEB_PUBLIC / "favicon.svg", mark_svg)

    banner = build_banner()
    write(MEDIA / "fabricator-banner.svg", banner.to_svg(scale=4, title="Fabricator", background=INK))

    social_svg = MEDIA / "fabricator-social.svg"
    write(social_svg, build_social().to_svg(scale=8, title="Fabricator", background=INK))

    tmp_mark = MEDIA / ".mark-tmp.svg"
    tmp_mark.write_text(mark_svg)
    rasterise(social_svg, MEDIA / "fabricator-social.png", 1280, 640)
    rasterise(social_svg, WEB_PUBLIC / "og.png", 1280, 640)
    rasterise(tmp_mark, WEB_PUBLIC / "favicon-32.png", 32, 32)
    rasterise(tmp_mark, WEB_PUBLIC / "apple-touch-icon.png", 180, 180)
    tmp_mark.unlink()
    print("done")


if __name__ == "__main__":
    main()
