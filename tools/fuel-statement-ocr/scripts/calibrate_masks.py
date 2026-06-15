"""Rough mask calibration: find blue-ink clusters on fuel statement scans."""
from __future__ import annotations

import json
import sys
from pathlib import Path

import numpy as np
from PIL import Image


def blue_mask(arr: np.ndarray) -> np.ndarray:
    r, g, b = arr[..., 0], arr[..., 1], arr[..., 2]
    return (b > 80) & (b > r + 30) & (b > g + 20) & (r < 200)


def main() -> None:
    scans = Path(sys.argv[1] if len(sys.argv) > 1 else r"F:\tl\scans")
    for path in sorted(scans.glob("ведомости.*.jpg")):
        im = Image.open(path).convert("RGB")
        w, h = im.size
        arr = np.asarray(im)
        m = blue_mask(arr)
        ys, xs = np.where(m)
        if len(xs) == 0:
            print(path.name, "no blue ink")
            continue
        print(
            path.name,
            f"size={w}x{h}",
            f"blue_bbox=({xs.min()},{ys.min()})-({xs.max()},{ys.max()})",
            f"norm=({xs.min()/w:.3f},{ys.min()/h:.3f},{(xs.max()-xs.min())/w:.3f},{(ys.max()-ys.min())/h:.3f})",
        )


if __name__ == "__main__":
    main()
