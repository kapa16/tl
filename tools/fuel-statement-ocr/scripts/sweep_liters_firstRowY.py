#!/usr/bin/env python3
"""Sweep quantity_liters.x and table.firstRowY; report row1 quantity_liters vs expected 102."""
from __future__ import annotations

import json
import shutil
import subprocess
import sys
from pathlib import Path

ROOT = Path(r"F:\tl\tools\fuel-statement-ocr")
TEMPLATE = ROOT / "internal" / "mask" / "templates" / "prihodnaya.json"
EXE = ROOT / "dist" / "FuelStatementOCR.exe"
GO = Path(r"F:\tl\tools\go-portable\go\bin\go.exe")
SCANS = Path(r"F:\tl\scans")
EXPECTED = "102"

LITERS_X_VALUES = [0.46, 0.48, 0.50, 0.52, 0.54]
FIRST_ROW_Y_VALUES = [0.198, 0.205, 0.212]
KG_X_FIXED = 0.68


def find_image() -> Path:
    matches = sorted(SCANS.glob("*.4.jpg"))
    if not matches:
        raise SystemExit(f"no *.4.jpg in {SCANS}")
    return matches[0]


def rebuild_exe() -> None:
    go = GO if GO.is_file() else None
    if go is None:
        import shutil as sh
        go_path = sh.which("go")
        if not go_path:
            raise SystemExit("go not found: install Go or tools/go-portable")
        go = Path(go_path)
    proc = subprocess.run(
        [str(go), "build", "-o", str(EXE), "./cmd/fuel-statement-ocr"],
        cwd=ROOT,
        capture_output=True,
        text=True,
        encoding="utf-8",
    )
    if proc.returncode != 0:
        raise SystemExit(f"go build failed:\n{proc.stderr}")


def patch_template(data: dict, liters_x: float, first_row_y: float) -> None:
    data["table"]["firstRowY"] = first_row_y
    found_liters = found_kg = False
    for col in data["table"]["columns"]:
        if col.get("id") == "quantity_liters":
            col["x"] = liters_x
            found_liters = True
        elif col.get("id") == "quantity_kg":
            col["x"] = KG_X_FIXED
            found_kg = True
    if not found_liters:
        raise KeyError("quantity_liters column not found")
    if not found_kg:
        raise KeyError("quantity_kg column not found")


def parse_row1_liters(stdout: str) -> tuple[str | None, float | None]:
    data = json.loads(stdout)
    for row in data.get("rows", []):
        if row.get("rowIndex") == 1:
            field = row.get("fields", {}).get("quantity_liters", {})
            return field.get("valueString"), field.get("confidence")
    return None, None


def numeric_distance(value_string: str | None, expected: str) -> float:
    if not value_string:
        return float("inf")
    digits = "".join(c for c in value_string if c.isdigit())
    if not digits:
        return float("inf")
    try:
        return abs(int(digits) - int(expected))
    except ValueError:
        return float("inf")


def main() -> None:
    if not EXE.is_file():
        print(f"Missing exe: {EXE}", file=sys.stderr)
        sys.exit(1)
    if not TEMPLATE.is_file():
        print(f"Missing template: {TEMPLATE}", file=sys.stderr)
        sys.exit(1)

    image = find_image()
    original_text = TEMPLATE.read_text(encoding="utf-8")
    backup = TEMPLATE.with_suffix(".json.bak-sweep-liters")
    shutil.copy2(TEMPLATE, backup)

    results: list[dict] = []
    try:
        for first_row_y in FIRST_ROW_Y_VALUES:
            for liters_x in LITERS_X_VALUES:
                data = json.loads(original_text)
                patch_template(data, liters_x, first_row_y)
                TEMPLATE.write_text(
                    json.dumps(data, ensure_ascii=False, indent=2) + "\n",
                    encoding="utf-8",
                )
                rebuild_exe()

                proc = subprocess.run(
                    [str(EXE), "--type=prihodnaya", str(image)],
                    capture_output=True,
                    text=True,
                    encoding="utf-8",
                )
                entry = {
                    "liters_x": liters_x,
                    "firstRowY": first_row_y,
                    "kg_x": KG_X_FIXED,
                    "returncode": proc.returncode,
                    "valueString": None,
                    "confidence": None,
                    "stderr": "",
                }
                if proc.returncode == 0 and proc.stdout.strip():
                    vs, conf = parse_row1_liters(proc.stdout)
                    entry["valueString"] = vs
                    entry["confidence"] = conf
                else:
                    entry["stderr"] = (proc.stderr or proc.stdout or "").strip()[:500]
                entry["distance"] = numeric_distance(entry["valueString"], EXPECTED)
                results.append(entry)
                print(
                    f"liters_x={liters_x:.2f} firstRowY={first_row_y:.3f} -> "
                    f"valueString={entry['valueString']!r} conf={entry['confidence']} "
                    f"dist={entry['distance']}"
                )
    finally:
        TEMPLATE.write_text(original_text, encoding="utf-8")
        if backup.is_file():
            backup.unlink()

    results.sort(key=lambda r: (r["distance"], -(r["confidence"] or 0)))
    print()
    print(f"Image: {image}")
    print(f"kg.x fixed at {KG_X_FIXED}")
    print(f"Expected row1 quantity_liters valueString: {EXPECTED!r}")
    print("=== Top 5 combinations (closest numeric match) ===")
    for r in results[:5]:
        print(
            f"  liters_x={r['liters_x']:.2f} firstRowY={r['firstRowY']:.3f} "
            f"valueString={r['valueString']!r} confidence={r['confidence']} distance={r['distance']}"
        )
    exact = [r for r in results if r["valueString"] == EXPECTED]
    if exact:
        print()
        print("=== Exact string match ===")
        for r in exact:
            print(
                f"  liters_x={r['liters_x']:.2f} firstRowY={r['firstRowY']:.3f} confidence={r['confidence']}"
            )
    else:
        best = results[0]
        print()
        print(
            f"Best: liters_x={best['liters_x']:.2f} firstRowY={best['firstRowY']:.3f} "
            f"valueString={best['valueString']!r} confidence={best['confidence']}"
        )


if __name__ == "__main__":
    main()
