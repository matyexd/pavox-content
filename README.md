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
- uses a generic `TelegramResponse[T]` type for Telegram JSON responses;
- groups updates from the same response into logical posts;
- has unit tests for text posts, albums, and mixed post ordering.

Media grouping is implemented and tested as a pure transformation, but it is not yet connected to the main polling loop. The running application still processes each raw Telegram update separately.

## Requirements

- Go 1.27.1 or newer compatible version;
- a Telegram bot;
- a Telegram channel where the bot is an administrator.

## Configuration

Create a local `.env` file in the project root:

```sh
export TELEGRAM_BOT_TOKEN='your-telegram-bot-token'
```

Load it into the current shell session:

```sh
source .env
```

The `.env` file is ignored by Git. Never commit a real bot token. If a token is exposed, revoke it through BotFather and generate a new one.

## Running the service

From the project root:

```sh
source .env
go run .
```

Publish a new post in the configured Telegram channel while the service is running. Photo files are currently saved in the project root using their Telegram `file_unique_id` as the filename.

Stop the service with `Ctrl+C`.

## Running tests

```sh
go test -v ./...
```

The current tests cover:

- normalization of a single text update;
- grouping an album into one logical post;
- choosing the largest photo size;
- preserving the order of mixed standalone posts and albums.

## Project structure

```text
main.go              Long-polling loop and offset coordination
telegram_api.go      Telegram-specific API methods
telegram_types.go    Telegram DTOs and normalized post types
processor.go         Update processing and grouping
processor_test.go    Unit tests for post grouping
utils.go             Shared HTTP and JSON request helpers
docs/                 Technical notes and implementation roadmap
```

## Known limitations

- media groups are only assembled within one `getUpdates` response;
- media grouping is not yet used by the main processing loop;
- the offset is stored only in memory;
- transient request failures stop the service;
- retries and exponential backoff are not implemented;
- downloaded files are written directly to the project root;
- the default HTTP client is used without application-level context or timeout management;
- LLM and Strapi integrations are not implemented yet.

For Telegram-specific implementation details and the roadmap, see [docs/TELEGRAM-BOT.md](docs/TELEGRAM-BOT.md).
