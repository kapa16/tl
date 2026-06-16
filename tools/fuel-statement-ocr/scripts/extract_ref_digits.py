#!/usr/bin/env python3
"""Bootstrap: extract digit PNGs from reference scan for go:embed."""
import json
import math
import sys
from pathlib import Path

try:
    from PIL import Image
except ImportError:
    sys.exit("pip install Pillow")

ROOT = Path(__file__).resolve().parents[1]
TEMPLATE = ROOT / "internal/mask/templates/prihodnaya.json"
OUT = ROOT / "internal/recognize/refdigits/prihodnaya"


def rotate90_cw(img, times):
    times = times % 4
    if times == 0:
        return img
    return img.transpose([None, Image.ROTATE_270, Image.ROTATE_180, Image.ROTATE_90][times])


def to_gray(img):
    return img.convert("L")


def letterbox(img, ref_w, ref_h):
    sw, sh = img.size
    scale = min(ref_w / sw, ref_h / sh)
    nw, nh = int(round(sw * scale)), int(round(sh * scale))
    scaled = img.resize((nw, nh), Image.BILINEAR)
    canvas = Image.new("RGB", (ref_w, ref_h), (255, 255, 255))
    ox, oy = (ref_w - nw) // 2, (ref_h - nh) // 2
    canvas.paste(scaled, (ox, oy))
    sx, sy = nw / ref_w, nh / ref_h
    dx, dy = ox / ref_w, oy / ref_h
    return canvas, dx, dy, sx, sy


def adjust_rect(r, dx, dy, sx, sy):
    return {
        "x": dx + r["x"] * sx,
        "y": dy + r["y"] * sy,
        "w": r["w"] * sx,
        "h": r["h"] * sy,
    }


def normalize_digit(gray, x0, y0, x1, y1, tw=24, th=32):
    crop = gray.crop((x0, y0, x1, y1))
    # Otsu-like threshold
    hist = crop.histogram()
    total = crop.size[0] * crop.size[1]
    sum_all = sum(i * hist[i] for i in range(256))
    sum_b, w_b, max_var, thresh = 0, 0, 0, 128
    for t in range(256):
        w_b += hist[t]
        if w_b == 0:
            continue
        w_f = total - w_b
        if w_f == 0:
            break
        sum_b += t * hist[t]
        m_b = sum_b / w_b
        m_f = (sum_all - sum_b) / w_f
        var_between = w_b * w_f * (m_b - m_f) ** 2
        if var_between > max_var:
            max_var = var_between
            thresh = t
    bin = crop.point(lambda p: 255 if p < thresh else 0, mode="L")
    inv = bin.point(lambda p: 255 - p, mode="L")
    return inv.resize((tw, th), Image.BILINEAR)


def main():
    scan = sys.argv[1] if len(sys.argv) > 1 else str(Path("F:/tl/scans").glob("*.4.jpg").__iter__().__next__() if False else "")
    if len(sys.argv) < 2:
        scans = list(Path("F:/tl/scans").glob("*.4.jpg"))
        if not scans:
            sys.exit("no scan")
        scan = str(scans[0])
    tmpl = json.loads(TEMPLATE.read_text(encoding="utf-8"))
    ref_w = tmpl["referenceSize"]["width"]
    ref_h = tmpl["referenceSize"]["height"]
    img = Image.open(scan)
    img = rotate90_cw(img, 1)  # 90 CW for ved4
    canvas, dx, dy, sx, sy = letterbox(img, ref_w, ref_h)
    gray = to_gray(canvas)
    w, h = gray.size
    OUT.mkdir(parents=True, exist_ok=True)
    for cell in tmpl["digitReference"]["cells"]:
        r = adjust_rect(cell, dx, dy, sx, sy)
        x0 = int(r["x"] * w)
        y0 = int(r["y"] * h)
        x1 = int((r["x"] + r["w"]) * w)
        y1 = int((r["y"] + r["h"]) * h)
        digit = cell["digit"]
        out = normalize_digit(gray, x0, y0, x1, y1)
        path = OUT / f"{digit}.png"
        out.save(path)
        print("wrote", path)


if __name__ == "__main__":
    main()
