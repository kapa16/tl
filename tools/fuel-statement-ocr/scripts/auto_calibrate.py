"""Auto-calibrate mask JSON from handwritten scan (blue ink clusters)."""
from __future__ import annotations

import json
from pathlib import Path

import numpy as np
from PIL import Image


def blue_mask(arr: np.ndarray) -> np.ndarray:
    r, g, b = arr[..., 0], arr[..., 1], arr[..., 2]
    return (b > 80) & (b > r + 30) & (b > g + 20) & (r < 200)


def cluster_x(xs: np.ndarray, w: int, min_gap: int = 40) -> list[tuple[float, float]]:
    if len(xs) == 0:
        return []
    xs = np.sort(xs)
    groups: list[list[int]] = [[int(xs[0])]]
    for x in xs[1:]:
        if x - groups[-1][-1] > min_gap:
            groups.append([int(x)])
        else:
            groups[-1].append(int(x))
    return [(min(g) / w, max(g) / w) for g in groups if len(g) > 5]


def main() -> None:
    scans = {
        "perelivnaya": Path(r"F:\tl\scans\ведомости.1.jpg"),
        "prihodnaya": Path(r"F:\tl\scans\ведомости.4.jpg"),
    }
    for typ, path in scans.items():
        im = Image.open(path).convert("RGB")
        w, h = im.size
        m = blue_mask(np.asarray(im))
        ys, xs = np.where(m)
        # date region
        dsel = (ys >= int(h * 0.07)) & (ys <= int(h * 0.14))
        dclusters = cluster_x(xs[dsel], w, 25)
        print(typ, "date clusters", dclusters[:6])
        for row in range(1, 4):
            y0, y1 = int(h * (0.205 + (row - 1) * 0.0375)), int(h * (0.205 + row * 0.0375))
            rsel = (ys >= y0) & (ys < y1)
            clusters = cluster_x(xs[rsel], w, 30)
            print(f"  row{row}", clusters)


if __name__ == "__main__":
    main()
