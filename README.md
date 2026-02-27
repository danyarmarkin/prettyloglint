# prettyloglint

Линтер для проверки стиля логов в коде. 
Позволяет выявлять и исправлять нарушения стиля логов, 
обеспечивая единообразие и читаемость логов в проекте.

### Поддерживаемые логерные библиотеки:
- `log/slog`
- `go.uber.org/zap` (в том числе с поддержкой полей `zap.Any`, `zap.String`, `zap.Int` и т.д.)

## Установка
1. В своем проекте создайте файл `.custom-gcl.yml`.
2. Добавьте в него следующее содержимое:
```yaml
version: ${GOLANGCI-LINT_VERSION}
plugins:
  - module: "github.com/danyarmarkin/prettyloglint"
    import: "github.com/danyarmarkin/prettyloglint/plugin"
    version: ${VERSION}
```
**Параметры**

| Переменная              | Описание                                                                               |
|-------------------------|----------------------------------------------------------------------------------------|
| `GOLANGCI-LINT_VERSION` | Версия golangci-lint, которую вы используете в своем проекте                           |
| `VERSION`               | Версия плагина, которую вы хотите использовать (`latest` чтобы использовать последнюю) |

3. Соберите кастомный бинарь golangci-lint с помощью команды:
```bash
golangci-lint custom
```

4. В конфигурационном файле `.golangci.yaml` укажите:
```yaml
linters-settings:
  custom:
    prettyloglint:
      type: "module"
linters:
  - enable:
      - prettyloglint
```
**Настройки**

| Параметр                    | Описание                                                                                    |
|-----------------------------|---------------------------------------------------------------------------------------------|
| `allowed-punctuation`       | [Optional] Разрешенные знаки препинания в логах (`default=",-/:()"`)                        |
| `ignore-zap-fields`         | [Optional] Игнорировать ли поля zap в логах (`default=false`)                               |
| `custom-sensitive-patterns` | [Optional] Пользовательские шаблоны для поиска чувствительных данных в логах (`default=[]`) |

**Пример**

```yaml
linters-settings:
  custom:
    prettyloglint:
      type: "module"
      settings:
        allowed-punctuation: ",. ()!"
        ignore-zap-fields: true
        custom-sensitive-patterns:
          - "username"
          - "callback_data"
linters:
  - enable:
      - prettyloglint
```

## Использование
- После установки плагина, вы можете использовать его вместе с golangci-lint для проверки стиля логов в вашем коде. Запустите команду:
```bash
custom-gcl run --config=.golangci.yaml
```

- **QuickFixes**: Если линтер обнаруживает нарушение стиля логов, он может предложить QuickFix для автоматического исправления проблемы. Вы можете применить эти исправления с помощью флага:
```bash
custom-gcl run --config=.golangci.yaml --fix
```

## Примеры использования
Можно найти в [testdata](integration_tests/testdata)