# User Rules — проект goskey

Дополняют `AGENTS.md`. При конфликте по проектной политике — приоритет у этого файла.

## Контекст репозитория

- **Исходники расширения:** `src/cfe/` (CFE `Атжн_Госключ`, префикс объектов `Атжн_`).
- **Назначение:** интеграция с порталом Госключ (HTTP, статусы, подписание).
- **Параметры генерации и ИБ:** только `.dev.env` (не дублировать в других файлах).

## Версия расширения (обязательно)

При **любой** доставляемой доработке в `src/cfe/` агент **обязан** увеличить `<Version>` в `src/cfe/Configuration.xml` по [std483](https://its.1c.ru/db/v8std/content/483/hdoc) (формат `Р.П.З.С`) и классифицировать выпуск по [std484](https://its.1c.ru/db/v8std/content/484/hdoc):

| Характер изменения | Что увеличивать |
|--------------------|-----------------|
| Исправление ошибки, мелкая правка без новой функциональности | **С** (сборка), напр. `1.3.2.5` → `1.3.2.6` |
| Новая функциональность, новые объекты метаданных | **З** (версия), сборку сбросить в `1`, напр. `1.3.3.1` |
| Крупный функциональный блок / подредакция | **П**, напр. `1.4.1.1` |
| Несовместимые изменения структуры данных | **Р** (редкость для расширения) |

Исключения: правки только в документации/правилах без изменений `src/cfe/`. Подробности — `.cursor/rules/extension-versioning.mdc` (правило всегда активно).

## Политика метаданных

- `NEW_OBJECTS_IN=extension` — новые объекты только в расширении.
- `EXTENSION_NAME=Атжн_Госключ`, `EXPORT_PATH=src/cfe`.
- Изменения XML/форм/ролей в рамках feature-dev выполняет агент (`1c-metadata-manager` или скиллы `meta-*`, `cfe-*`, `form-*`), а не «только пользователь вручную».

## Feature-dev workflow (канонический вход)

Главный процесс доработок: скилл **`1c-feature-dev`** или команда **`/1c-feature-dev`**.

Скилл уже использует канонические имена агентов goskey напрямую — отдельные legacy-агенты `1c-code-*` удалены как дубли. Соответствие исходным именам из [1c-ai-feature-dev-workflow](https://github.com/AndreevED/1c-ai-feature-dev-workflow) (если встретишь их в сторонних материалах):

| Оригинал (AndreevED) | Субагент / инструмент в goskey |
|----------------------|--------------------------------|
| `1c-code-explorer` | `1c-explorer` |
| `1c-code-architect` | `1c-architect` (план) / `1c-arch-reviewer` (ревью архитектуры) |
| `1c-code-writer` | `1c-developer` |
| `1c-code-reviewer` | MCP `check_1c_code` + `review_1c_code`; субагент `1c-code-reviewer` — только по явной просьбе пользователя |
| `1c-code-simplifier` | `1c-refactoring`, опционально после Phase 6 по запросу |
| Правила `1c-rules.md` / `~/.claude/rules/` | `AGENTS.md` + `.cursor/rules/coding-standards.mdc` и on-demand-правила по задаче |

### Скиллы cc-1c-skills в фазах feature-dev

| Тип этапа в плане | Скилл / команда |
|-------------------|-----------------|
| Новый объект метаданных | `meta-compile`, `meta-edit`, `meta-validate` или `1c-metadata-manage` |
| Форма | `form-add`, `form-edit`, `form-compile`, `form-validate` |
| Расширение / заимствование | `cfe-borrow`, `cfe-patch-method` |
| СКД / отчёт | `skd-compile`, `skd-edit` |
| Роль | `role-compile`, `role-validate` |
| Загрузка в ИБ / UI-тест | `/deploy-and-test`, `web-test`, `1c-tester` |

Предпочитай скиллы из **`.cursor/skills/<name>/`** (cc-1c-skills). Дублирующие скрипты внутри `1c-metadata-manage/tools/` — только если скилл явно отсылает туда.

### Два pipeline — когда какой

| Ситуация | Pipeline |
|----------|----------|
| Новая доработка, неясные требования, нужен план с этапами и артефактами | **feature-dev** (фазы 0–8, `.tasks/`) |
| Задача уже формализована (OpenSpec, короткое ТЗ), &lt; ~20 строк, один модуль | **quick-fix** или `subagent-pipeline` из `AGENTS.md` без полного feature-dev |
| Крупная фича с контрактом поведения | OpenSpec (`/opsx:*`) **+** feature-dev; спецификация в `openspec/`, ход работы в `.tasks/` |

## MCP и индексация

Перед Phase 2 убедись, что доступны MCP из `.cursor/mcp.json` (`/checkmcp`, `/doctor`). Исследование кода — цепочка из `mcp-1c-tools` (graph → code-metadata), не слепой Grep по умолчанию.

## Migrated content from a previous setup

<!-- start of migrated content -->
<!-- end of migrated content -->
