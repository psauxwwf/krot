# Changelog

## xray-checker как обычная зависимость (fork)

Проблема: `xray-checker` клонировался отдельным репозиторием в `./xray-checker`,
после чего `Taskfile.yml` переписывал `module xray-checker` →
`module github.com/kutovoys/xray-checker` и все внутренние импорты через `sed`,
чтобы Go принимал модуль. Корневая причина — в upstream `go.mod` задан неверный
путь модуля (`module xray-checker`), а также присутствуют файлы с emoji в именах,
из-за чего Go не может создать валидный zip модуля и опубликовать его в proxy.

Решение: создан fork `github.com/psauxwwf/xray-checker`, в котором исправлен путь
модуля и импорты, а также удалены emoji-файлы. Fork помечен тегом `v1.3.3`.

### Изменения в krot

- `go.mod`: require обновлён до `v1.3.3`; `replace` теперь указывает на fork —
  `replace github.com/kutovoys/xray-checker => github.com/psauxwwf/xray-checker v1.3.3`.
  Локальный клон больше не нужен.
- `go.sum`: обновлён контрольными суммами fork-модуля.
- `Taskfile.yml`: удалена задача `setup:xray-checker` (клонирование + sed-перезапись)
  и её вызов из `setup`.
- `.gitignore`: удалена строка `/xray-checker`.
- Локальный клон `./xray-checker` удалён; сборка разрешает зависимость из кэша
  модулей Go.

### Изменения в fork (github.com/psauxwwf/xray-checker, тег v1.3.3)

- `go.mod`: `module xray-checker` → `module github.com/kutovoys/xray-checker`.
- Все импорты `"xray-checker/..."` переписаны на полный GitHub-путь (17 файлов).
- Удалены `.github/ISSUE_TEMPLATE/🐞-bug-report.md` и
  `.github/ISSUE_TEMPLATE/💡-feature-request.md` (emoji в именах ломает zip модуля).

## Убраны именованные импорты xray-checker

В `pkg/checker/checker.go` убраны псевдонимы импортов (`checkerlogger`,
`checkermodels`, `checkersubscription`, `checkerxray`) — теперь используются
обычные имена пакетов (`logger`, `models`, `subscription`, `xray`). Все
  места использования обновлены. Поведение не изменилось; `gofmt`, `go build`,
  `go vet` — чисто.

## Синхронизация тегов fork (github.com/psauxwwf/xray-checker)

Fork `psauxwwf/xray-checker` содержал битый промежуточный тег `v1.3.2` (коммит
`17c855e` — исправлен путь модуля, но ещё остались emoji-файлы, из-за чего Go не
может собрать zip модуля и тег непригоден для использования). Тег `v1.3.2` удалён
через GitHub API. Теперь набор тегов fork синхронизирован: он содержит все теги
upstream (`v0.1.0`–`v1.3.1`, на тех же коммитах) плюс единственный чистый
тег-исправление `v1.3.3`, на который и ссылается `go.mod` проекта. В `go.sum`
тег `v1.3.2` не фигурировал, поэтому удаление безопасно; исходный код krot не
потребовал изменений.
