# Telegram Scraper

A Go application to scrape Telegram web.

## Structure

- `cmd/scraper/main.go`: Main application entry point.
- `internal/`: Private application and library code.
- `pkg/`: Public library code.
  - `config/`: Configuration loading.
  - `scraper/`: Scraping logic.
  - `storage/`: Data storage.
  - `telegram/`: Telegram client.
- `configs/`: Configuration files.
- `scripts/`: Helper scripts.

## Usage

```bash
go run main.go
```
