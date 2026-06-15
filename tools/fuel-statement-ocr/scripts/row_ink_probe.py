"""Find blue-ink cell centers per row for mask tuning."""
from __future__ import annotations

from pathlib import Path

import numpy as np
from PIL import Image


def blue_mask(arr: np.ndarray) -> np.ndarray:
    r, g, b = arr[..., 0], arr[..., 1], arr[..., 2]
    return (b > 80) & (b > r + 30) & (b > g + 20) & (r < 200)


def main() -> None:
    path = Path(r"F:\tl\scans\ведомости.4.jpg")
    im = Image.open(path).convert("RGB")
    w, h = im.size
    arr = np.asarray(im)
    m = blue_mask(arr)
    ys, xs = np.where(m)
    for row in range(16):
        y0 = int(h * (0.205 + row * 0.0375))
        y1 = int(h * (0.205 + (row + 1) * 0.0375))
        sel = (ys >= y0) & (ys < y1)
        if not np.any(sel):
            continue
        rx, ry = xs[sel], ys[sel]
        print(f"row {row+1}: y={y0}-{y1} x_range={rx.min()}-{rx.max()} norm_x={rx.min()/w:.3f}-{rx.max()/w:.3f} count={len(rx)}")


if __name__ == "__main__":
    main()
