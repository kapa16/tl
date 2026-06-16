# FuelStatementOCR

Утилита распознавания рукописных цифр на сканах заправочных ведомостей (Go, template matching по полосе `1234567890`).

**Версия пайплайна v2:** EXIF → автоориентация (0/90/180/270°) → letterbox к эталону 2480×3507 → `AdjustTemplateForCanvas` → распознавание по layout.

Сдвиг изображения по полосе `1234567890` отключён: координаты маски откалиброваны под letterbox (см. `prihodnaya.json`, сетка на overlay).

Выбранный поворот — в JSON `layout.orientationApplied`; при близких оценках приоритет у варианта с лучшим совпадением полосы цифр.

## Сборка

```powershell
.\build.ps1
```

Требуется Go 1.22+ (в репозитории: `tools/go-portable/` или системный `go`).

Результат: `dist/FuelStatementOCR.exe` → скопировать в макет EPF `КомпонентаOCR/Ext/Template.bin`.

## Вызов

```text
FuelStatementOCR.exe --type=prihodnaya [--dump-crops dir] [--dump-layout dir] [--dump-ref dir] "path\to\scan.jpg"
```

Флаги `--type`, `--dump-crops`, `--dump-layout`, `--dump-ref` должны идти **до** пути к изображению.

### Отладка

```text
FuelStatementOCR.exe --type=prihodnaya scan.jpg --dump-crops .\dump --dump-layout .\dump
```

- `--dump-crops` — PNG-вырезки маски чернил по строкам и колонкам (`row1_quantity_liters.png`, …)
- `--dump-layout` — `layout_overlay.png` с полосами строк (красный) и колонок (зелёный)

- `--dump-ref` — PNG шаблонов цифр 0–9, используемых при распознавании

Stdout — JSON (UTF-8), код возврата 0 при успешном разборе (в т.ч. частичном).

## Шаблоны цифр

- Встроенные PNG: `internal/recognize/refdigits/prihodnaya/` (`go:embed`) — segment-style с полосы `1234567890`
- Перегенерация с эталонного скана:

```powershell
go run ./cmd/extract-ref-digits --type=prihodnaya --out=internal/recognize/refdigits/prihodnaya scans/ведомости.4.jpg
.\build.ps1
```

- Поиск полосы: 2D-скан верхней зоны (`ScanReferenceStrip`), оценка `StripTemplateCoverage` (доля годных segment-шаблонов 0–9), отсев ложных 10-пиков без регулярности
- Ориентация: выбор поворота по `stripCov` на letterbox-канвасе (не по сырому кадру)
- Matching рукописи: корреляция + mask overlap + **5-band segment signature** + штрафы за «палку»/ложную 7
- Фильтр шаблонов-палок: `horizontalInkSpread` + aspect ratio; для segment-цифр — `SegmentTemplateOK`
- Строки таблицы: сегментация ink-runs внутри колонки → fallback на сетку 5 ячеек
- Чернила: синие и чёрные/серые оттенки — `IsInkPixel` (хрома B-канала + локальный контраст), `InkMaskFull` = цвет + adaptive на gray

## Типы бланков

| `--type` | Enum 1С | Эталон |
|----------|---------|--------|
| `perelivnaya` | Переливная | `scans/ведомости.1.jpg` |
| `prihodnaya` | Приходная | `scans/ведомости.4.jpg` |
| `zapravka` | Заправочная | `scans/ведомости.3.jpg` |

## JSON v1.1

Поля `header`, `footer`, `rows` — без изменений для BSL. Дополнительно:

```json
{
  "version": "1.1",
  "layout": {
    "orientationApplied": 180,
    "exifOrientation": 1,
    "homographyConfidence": 0.85,
    "tableFound": true,
    "columns": [{ "id": "quantity_liters", "x0": 0.46, "x1": 0.58, "source": "fallback" }],
    "rowBands": [{ "rowIndex": 1, "y0": 0.205, "y1": 0.242 }]
  }
}
```

## Калибровка

- Маски: `internal/mask/templates/*.json` (`fallbackX0`/`fallbackX1`, `headerBand`, `rowHeight`, `firstRowY`)
- Перебор координат: `scripts/sweep_kg_firstRowY.py`, `scripts/sweep_liters_firstRowY.py` (требуют `go build` после правки JSON — `go:embed`)
- Прогон эталонов: `scripts/verify_scan.ps1`
- Отладка кропов: `--dump-crops`, `--dump-layout`

**Приходная (`ведомости.4.jpg`):** полоса `1234567890` — `referenceDigits.confidence` ~0.94 (10/10 segment-шаблонов); ориентация 180°. Распознавание строк — в работе (геометрия колонок / сетка ячеек).
