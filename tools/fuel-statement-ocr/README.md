# FuelStatementOCR

Утилита распознавания рукописных цифр на сканах заправочных ведомостей (Go, template matching по полосе `1234567890`).

## Сборка

```powershell
.\build.ps1
```

Требуется Go 1.22+ (в репозитории: `tools/go-portable/` или системный `go`).

Результат: `dist/FuelStatementOCR.exe` → копировать в макет EPF `КомпонентаOCR/Ext/Template.bin`.

## Вызов

```text
FuelStatementOCR.exe --type=perelivnaya "path\to\scan.jpg"
FuelStatementOCR.exe --type=prihodnaya "path\to\scan.jpg"
FuelStatementOCR.exe --type=zapravka "path\to\scan.jpg"
```

Stdout — JSON (UTF-8), код возврата 0 при успешном разборе (в т.ч. частичном).

## Типы бланков

| `--type` | Enum 1С | Эталон |
|----------|---------|--------|
| `perelivnaya` | Переливная | `scans/ведомости.1.jpg` |
| `prihodnaya` | Приходная | `scans/ведомости.4.jpg` |
| `zapravka` | Заправочная | `scans/ведомости.3.jpg` |

## Калибровка масок

Скрипты в `scripts/`; маски в `internal/mask/templates/*.json`.

Точная калибровка — итеративно через `--dump-crops` (планируется) или правку `columnRanges` в `internal/recognize/rowscan.go`.
