# Pavox Content

Pavox Content is a Go service for importing content from a Telegram channel into the Pavox cosmetics e-commerce workflow.

The project is also a hands-on environment for learning Go and backend engineering. Its goal is not only to automate content processing, but to make every part of the pipeline understandable and maintainable.

The planned pipeline is:

```text
Telegram channel
        ↓
Telegram Bot API
        ↓
Go service
        ↓
Post normalization
        ↓
LLM classification and structured extraction
        ↓
Validation
        ↓
Strapi drafts
        ↓
Pavox website
```

## Current status

The application currently:

- receives channel posts through Telegram Bot API long polling;
- tracks Telegram updates with `offset`;
- reads the bot token from an environment variable;
- parses text posts, captions, photos, and media group identifiers;
- selects the largest available size of each photo;
- resolves Telegram `file_id` values through `getFile`;
- streams downloaded images to local files;
- sends each normalized post's `caption` and `text` to OpenAI Responses API;
- constrains the model output with a strict JSON Schema;
- saves the extracted product as `product.json` next to `post.json` and its images;
- uploads downloaded images to Strapi;
- creates and publishes products in Strapi;
- prevents duplicate products with a stable `telegramSourceId`;
- uses a generic `TelegramResponse[T]` type for Telegram JSON responses;
- groups updates from the same response into logical posts;
- has unit tests for text posts, albums, and mixed post ordering.

Each logical post is processed in its own timestamped directory under `downloads/`.

## Requirements

- Go 1.27.1 or newer compatible version;
- a Telegram bot;
- a Telegram channel where the bot is an administrator.
- an OpenAI API key.

## Configuration

Create a local `.env` file in the project root:

```sh
export TELEGRAM_BOT_TOKEN='your-telegram-bot-token'
export TELEGRAM_ALLOWED_CHANNEL_IDS='-1001234567890,-1009876543210'
export OPENAI_API_KEY='your-openai-api-key'
export STRAPI_BASE_URL='http://localhost:1337'
export STRAPI_API_TOKEN='your-strapi-api-token'
```

Set `TELEGRAM_ALLOWED_CHANNEL_IDS` to the comma-separated numeric IDs of production source channels. The service refuses to start if this setting is empty or invalid. Posts from other channels are acknowledged and skipped before file downloads, LLM extraction, notifications, or Strapi writes. Source IDs are saved as `chat_id` in `post.json`; albums are grouped separately per channel.

Load it into the current shell session:

```sh
source .env
```

The `.env` file is ignored by Git. Never commit Telegram or OpenAI credentials. Revoke and replace any exposed credential immediately.

## Running the service

From the project root:

```sh
source .env
go run .
```

Publish a new post in the configured Telegram channel while the service is running. The normalized post, downloaded photos, and extracted product JSON are saved in a timestamped directory under `downloads/`.

Stop the service with `Ctrl+C`.

## Running tests

```sh
go test -v ./...
```

The current tests cover:

- normalization of a single text update;
- grouping an album into one logical post;
- choosing the largest photo size;
- preserving the order of mixed standalone posts and albums;
- building the structured OpenAI request and parsing the product response.

## Project structure

```text
main.go              Long-polling loop and offset coordination
telegram_api.go      Telegram-specific API methods
telegram_types.go    Telegram DTOs and normalized post types
processor.go         Update processing and grouping
processor_test.go    Unit tests for post grouping
llm.go               OpenAI request construction and product extraction
product_schema.go    Structured Outputs JSON Schema
strapi.go            Strapi JSON API client
strapi_upload.go     Multipart media upload and cleanup
strapi_import.go     Product mapping and import orchestration
utils.go             Shared HTTP and JSON request helpers
docs/                 Technical notes and implementation roadmap
```

## Known limitations

- media groups are only assembled within one `getUpdates` response;
- the offset is stored only in memory;
- transient request failures stop the service;
- retries and exponential backoff are not implemented;
- the default HTTP client is used without application-level context or timeout management;
- a failed post retry can create another timestamped directory;
- uploaded files are buffered in memory one at a time before they are sent to Strapi;
- a publish failure leaves the draft and its media in Strapi so the next retry can finish publication.

For Telegram-specific implementation details and the roadmap, see [docs/TELEGRAM-BOT.md](docs/TELEGRAM-BOT.md).

## Telegram publication notifications

Set these optional environment variables and restart the service:

```sh
export TELEGRAM_LOG_CHAT_ID='-1001234567890'
export SITE_BASE_URL='https://your-site.example'
```

Add the bot to a separate group and send `/chatid@YourBotUsername` there
(or `/chatid` in a private conversation). The running bot replies with the
chat ID; copy it into `TELEGRAM_LOG_CHAT_ID`. Allow the bot to send messages.
Use a separate log group rather than the source channel.

Notifications cover receipt, processing, publication, failure, and skipped
non-product posts. Success links use `SITE_BASE_URL/product/{slug}`. Failure
messages identify the stage and include a truncated error with configured credentials redacted.
Notification delivery errors do not interrupt publication. Without
`TELEGRAM_LOG_CHAT_ID`, notifications are disabled. Failed posts are acknowledged without automatic retries so they cannot block later posts.
After fixing the cause, resend the failed post.

### Forum topics

Send `/chatid@YourBotUsername` inside the desired topic. The reply stays in
that topic and includes both the group ID and the topic ID. Configure:

```sh
export TELEGRAM_LOG_CHAT_ID='-1004315915630'
export TELEGRAM_LOG_THREAD_ID='your-topic-id'
```

Restart the bot after updating the environment. Leave `TELEGRAM_LOG_THREAD_ID`
unset for a normal chat or the default destination.

## Product variants import (v2)

Deploy the backend with `POST /api/internal/catalog/telegram-import` before this bot.
There is no fallback to legacy product price writes. For existing products run the
baseline catalog migration before resuming imports. The frontend v2 build flag
is independent of this importer endpoint.

Grant the bot's custom Strapi API token the `product / telegram-import / importProduct`
action (`api::product.telegram-import.importProduct`), product `find`, category
`find`, and the existing upload permissions. Public/Authenticated storefront roles
and read-only API tokens cannot use this import endpoint. Never expose it publicly
by disabling authentication.

Extraction includes `volume_ml` and `variants`: each size has its own `price`,
`weight_grams`, and `volume_ml`. Grams and millilitres are distinct; absent values
stay null. The prompt covers г/гр/g, мл/ml, explicit kilograms/litres, and excludes
ingredient quantities and application doses. Missing prices are never invented.
Top-level single-size fields remain for compatibility with saved extractions;
multiple sizes use the array. New writes send commercial data only inside variants.

The backend creates a product and its SKU(s) in one transaction, including v1
mapping for a newly priced product. Re-import of the SAME telegramSourceId updates
exact-size SKU prices and keeps SKU identity, availability, activation and editorial
content. It does not publish unrelated draft edits or upload photos again.
Absorbed legacy source cards resolve to their canonical product via migration mapping.
Existing unpublished products require review (import_product_unpublished_review).
The first SKU of a new product is active; additional sizes are saved inactive for
review/activation after the v1 compatibility window. Unknown legacy sizes require
an audited correction; the importer returns import_size_requires_review rather
than guessing or rewriting an existing SKU size. Existing unbaselined products
return import_requires_baseline_migration. Size-less or ambiguous extraction must
not overwrite a known-size price. A new post/source is not matched by product name.

Price updates replace price/currentPrice with the explicit RUB price and clear
oldPrice. Missing-price entries preserve existing prices; new entries without a
price do not create sellable SKUs. Already inactive SKUs are never reactivated.
Products with no priced SKU remain unavailable. After a timeout/error from the
transactional import, uploaded media are retained because the transaction may have
committed; remove confirmed orphaned uploads separately. No production notifications,
LLM calls or writes to a real Strapi are made by the normal test suite.

Run `go test -race ./...` and `go vet ./...`. Backend integration coverage is
`npm run test:telegram-import` in vox-backend (disposable SQLite; real PostgreSQL
concurrency remains a separate pre-release check).
