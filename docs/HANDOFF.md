# Handoff: Telegram → LLM → Strapi

Актуально на 11 октября 2026 года.

## Контекст работы

`pavox-content` — учебный и одновременно реальный Go-сервис для переноса товарных публикаций Pavox из Telegram в Strapi.

Целевая цепочка:

```text
Telegram channel
        ↓
Telegram Bot API
        ↓
Go service
        ↓
Группировка и нормализация публикации
        ↓
OpenAI Responses API + Structured Outputs
        ↓
Валидация ProductExtraction
        ↓
Загрузка изображений и создание draft в Strapi
```

Проект используется для изучения Go. Перед продолжением обязательно прочитать корневой `AGENTS.md`: предпочтительный процесс — объяснить концепцию, дать пользователю небольшую задачу, проверить его реализацию и только затем увеличивать помощь. Не переписывать код пользователя без прямого запроса.

## Что уже реализовано и закоммичено

Последние коммиты:

```text
1871ee8 test Telegram photo downloads
0961f54 process Telegram update batches
39e11ff process grouped Telegram posts
```

Работает следующее:

- long polling Telegram Bot API через `getUpdates`;
- чтение `TELEGRAM_BOT_TOKEN` из переменной окружения;
- generic-оболочка `TelegramResponse[T]`;
- группировка Telegram updates в логические `TelegramPost`;
- объединение альбомов по `media_group_id`;
- выбор наибольшего размера каждой фотографии;
- сохранение порядка одиночных публикаций и альбомов;
- скачивание файлов через интерфейс `TelegramFileClient`;
- обработка всей пачки до обновления `offset`;
- сохранение прежнего `offset`, если обработка публикации завершилась ошибкой;
- unit-тесты группировки, порядка, offset и скачивания фотографии.

`downloads/` игнорируется Git.

## Текущие незакоммиченные изменения

На момент создания документа `git status --short` показывает:

```text
 M processor.go
 M processor_test.go
 M telegram_types.go
?? docs/HANDOFF.md
?? llm.go
?? product_schema.go
?? product_types.go
```

### Каталог отдельной Telegram-публикации

Незакоммиченные изменения в `processor.go`, `processor_test.go` и `telegram_types.go` добавляют такую структуру:

```text
downloads/
└── 2026-10-11_20-23-34.571011000/
    ├── post.json
    ├── <telegram-file-unique-id-1>.jpg
    └── <telegram-file-unique-id-2>.jpg
```

Имя каталога создаётся через:

```go
time.Now().Format("2006-01-02_15-04-05.000000000")
```

`post.json` содержит нормализованный `TelegramPost`, а не сырой Telegram `Update`. Для полей `TelegramPost` добавлены JSON tags. Тест проверяет создание каталога, формат его имени, `post.json` и фотографию.

До появления `llm.go` эти изменения проходили `go test ./...`.

### Текущая ошибка сборки

Сейчас `go test ./...` не компилируется:

```text
responseLLMHandle redeclared in this block
```

Причина: пустая функция `responseLLMHandle` объявлена одновременно в:

- `product_types.go`;
- `llm.go`.

Первое действие следующего этапа — оставить эту функцию только в одном файле. Логичнее хранить обработку ответа в `llm.go`, а в `product_types.go` оставить только типы. Не удалять пользовательский код молча: сначала объяснить конфликт.

## Текущая модель результата LLM

В `product_types.go` создана структура:

```go
type ProductExtraction struct {
	Name              *string  `json:"name"`
	Description       *string  `json:"description"`
	Compound          *string  `json:"compound"`
	MethodApplication *string  `json:"method_application"`
	Price             *int     `json:"price"`
	WeightGrams       *int     `json:"weight_grams"`
	CategorySlugs     []string `json:"category_slugs"`
}
```

Семантика `Name`:

- `Name != nil` — публикация распознана как конкретный товар;
- `Name == nil` — это не товар, публикацию нужно пропустить;
- отсутствие цены, веса или способа применения само по себе не означает, что публикация не является товаром.

## JSON Schema: текущее состояние и обязательные исправления

`product_schema.go` содержит `productExtractionSchema() map[string]any`.

Уже сделано:

- все свойства перечислены в `required`, как требует Structured Outputs;
- nullable-поля используют union с `null`;
- у корневого объекта установлено `additionalProperties: false`.

Осталось исправить:

1. Go-тип `Name` уже изменён на `*string`, но schema пока всё ещё содержит только `"type": "string"`.
2. Для `name` нужно установить:

```json
{
  "type": ["string", "null"],
  "description": "Product name, or null when the publication does not describe a specific product."
}
```

3. `category_slugs.items` пока разрешает любую строку. Нужно добавить `enum` допустимых slug.

## Категории временного dev-каталога

По текущей dev-админке Strapi используется временный статический справочник:

| Название | Slug | Родитель |
|---|---|---|
| Для волос | `category` | — |
| Для лица и тела | `category-1` | — |
| Шампуни | `category-2` | `category` |
| Мужская косметика | `category-3` | — |
| ЭКО-бытовая химия | `category-4` | — |
| SOS-уход | `category-5` | — |
| Бальзамы и маски для волос | `category-6` | `category` |
| Несмываемый уход за волосами | `category-7` | `category` |

Предлагаемая Go-модель временного справочника:

```go
type ProductCategory struct {
	Name       string
	Slug       string
	ParentSlug string
}
```

Модель должна возвращать наиболее конкретный slug. Родительский slug добавляет Go после валидации. Например, для твёрдого бальзама модель возвращает только `category-6`, а Go добавляет `category`.

Для `category_slugs.items` нужен enum:

```json
{
  "type": "string",
  "enum": [
    "category",
    "category-1",
    "category-2",
    "category-3",
    "category-4",
    "category-5",
    "category-6",
    "category-7"
  ]
}
```

### Важное различие окружений

Локальная база `/Users/matvey/projects/vox-backend/.tmp/data.db` содержит другой, более новый каталог со slug вроде `volosy`, `shampuni` и `balzamy-i-kondicionery`. Dev-админка на скриншотах содержит временные slug `category-*`.

Не смешивать эти наборы. Текущий временный mapper выше предназначен для dev-окружения со slug `category-*`. Перед интеграцией со Strapi нужно явно определить URL целевого окружения.

В будущем статический mapper следует заменить загрузкой категорий из Strapi при старте сервиса с кэшированием и последующим обновлением по таймеру или webhook.

## Текущая инструкция модели

Рабочая developer-инструкция, проверенная в Playground:

```text
Сначала определи, описывает ли публикация конкретный товар.

Публикация считается товаром, если из текста можно определить конкретный продукт: его название или однозначный тип и характеристики. Цена, вес и способ применения могут отсутствовать — это не делает публикацию нетоварной.

Если публикация не похожа на описание конкретного товара, например это новость, поздравление, общий совет, анонс, личная история или реклама без определимого продукта:

- верни null в поле name;
- верни null во всех остальных строковых и числовых полях;
- верни пустой массив в category_slugs.

Если публикация описывает конкретный товар, поле name должно содержать его название и не должно быть null.

Извлеки данные товара из Telegram-публикации.

Не выдумывай отсутствующие значения. Если строковое или числовое значение отсутствует, верни null.

Определи основной товар, который продаётся или описывается в публикации. Не назначай категории по упоминаниям других товаров, сравнений или рекомендаций.

Для поля category_slugs выбери одну или несколько наиболее конкретных подходящих категорий только из списка:

- category — Для волос. Используй только для товаров для волос, которые нельзя отнести к более конкретной категории ниже.
- category-1 — Для лица и тела.
- category-2 — Шампуни.
- category-3 — Мужская косметика.
- category-4 — ЭКО-бытовая химия.
- category-5 — SOS-уход.
- category-6 — Бальзамы и маски для волос.
- category-7 — Несмываемый уход за волосами.

Правила выбора категорий:

- Возвращай только slug, без названия категории.
- Выбирай наиболее конкретную категорию.
- Если выбрана дочерняя категория для волос, не добавляй category: родительскую категорию Go добавит самостоятельно.
- Не выбирай категорию только потому, что соответствующее слово упоминается в тексте. Категория должна относиться именно к основному товару публикации.
- Если подходящей категории нет или тип товара невозможно определить, верни пустой массив.
- category_slugs всегда должен быть массивом, а не null.
```

### Наблюдение после теста Playground

Для публикации о твёрдом бальзаме модель правильно вернула:

```json
"category_slugs": ["category-6"]
```

Но она вывела предполагаемый способ применения, хотя полноценной инструкции в публикации не было. Инструкцию нужно уточнить: `method_application` заполняется только тогда, когда автор явно описал последовательность или способ применения; общие знания модели использовать нельзя. Аналогично `compound` должен отражать только явно названные ингредиенты, а не достраивать полный состав.

## Как должен выглядеть запрос OpenAI

Используется Responses API и модель `gpt-6-luna`.

Код, сгенерированный Playground, был изучен, но его нельзя переносить буквально. В `input` Playground добавил предыдущие `reasoning` и `assistant` items. Каждая Telegram-публикация должна обрабатываться независимым запросом.

В запросе оставить:

- один developer message с инструкцией;
- один user message с `caption` и `text` текущей публикации;
- `text.format` типа `json_schema`;
- `name: product_extraction`;
- `strict: true`;
- `reasoning.effort: low` на первом этапе;
- `store: false`.

На первом этапе не использовать:

- streaming;
- историю предыдущих ответов;
- `reasoning.encrypted_content`;
- reasoning summary;
- tools;
- web search;
- `service_tier`, если достаточно значения по умолчанию.

Не отправлять модели Telegram `file_id`, `update_ids`, размеры фотографий и `media_group_id`, если анализ изображений ещё не используется. Для текстового извлечения достаточно `caption` и `text`.

Официальный Go SDK: `github.com/openai/openai-go/v3`. Клиент читает `OPENAI_API_KEY` из окружения. Ключ нельзя записывать в код, `.env` под Git, логи или документацию.

Ожидаемый путь обработки:

```text
client.Responses.New(...)
        ↓
проверка ошибки и статуса Response
        ↓
response.OutputText()
        ↓
json.Unmarshal(..., &ProductExtraction)
        ↓
валидация результата
```

Также нужно обработать refusal, failed/incomplete response и отсутствие output text.

## Strapi: проверенный контракт товара

Схема находится в:

```text
/Users/matvey/projects/vox-backend/src/api/product/content-types/product/schema.json
```

Основные поля:

- обязательные: `name`, `slug`;
- необязательные: `description`, `media`, `price`, `categories`, `method_application`, `compound`, `weightGrams`;
- `draftAndPublish: true`.

Компонент цены:

```text
/Users/matvey/projects/vox-backend/src/components/shared/price.json
```

В нём используются `price`, `currentPrice`, `currency` и необязательный `oldPrice`. `slug` товара должен создавать Go, а не LLM. Изображения сначала загружаются в Strapi отдельно, затем их ID передаются при создании товара. Товар сначала создаётся как draft, автоматическая публикация не нужна.

## Рекомендуемый порядок продолжения

1. Объяснить и устранить дублирование `responseLLMHandle` между `llm.go` и `product_types.go`.
2. Добиться зелёного `go test ./...` без изменения поведения.
3. Исправить nullable `name` в `productExtractionSchema`.
4. Создать статический mapper категорий и генерировать из него:
   - текст списка для developer prompt;
   - enum допустимых slug для JSON Schema;
   - добавление родительского slug после ответа модели.
5. Добавить тесты schema/mapper без сетевых запросов.
6. Вынести developer-инструкцию в отдельную функцию или константу и добавить правило против выдумывания `method_application` и `compound`.
7. Подключить официальный OpenAI Go SDK и `OPENAI_API_KEY`.
8. Создать интерфейс LLM extractor, чтобы unit-тесты не обращались к реальному API.
9. Реализовать один нестриминговый запрос Responses API.
10. Распарсить output JSON в `ProductExtraction`.
11. Если `Name == nil`, считать публикацию нетоварной и не создавать Strapi product.
12. Валидировать категории по mapper и добавлять родительские категории.
13. Сохранять результат рядом с `post.json`, например как `product.json`, для отладки до интеграции Strapi.
14. Подключить extraction к `processPost` после сохранения исходного Telegram-поста и фотографий.
15. После стабилизации extraction реализовать Strapi-клиент, загрузку media и создание draft product.

## Проверки перед следующим коммитом

После устранения текущего конфликта выполнить:

```sh
gofmt -w *.go
GOCACHE=/tmp/pavox-content-go-build go test ./...
git diff --check
git status --short
```

Не включать содержимое `downloads/`, `.env`, Telegram token или OpenAI API key в коммит.
