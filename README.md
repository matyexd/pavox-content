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
export OPENAI_API_KEY='your-openai-api-key'
export STRAPI_BASE_URL='http://localhost:1337'
export STRAPI_API_TOKEN='your-strapi-api-token'
```

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
