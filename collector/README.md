# Telegram Collector

A Go application to collect data from Telegram.

## Structure

- `cmd/collector/main.go`: Main application entry point.
- `internal/`: Private application and library code.
- `pkg/`: Public library code.
  - `config/`: Configuration loading.
  - `collector/`: Data collection logic.
  - `storage/`: Data storage.
  - `telegram/`: Telegram client.
- `configs/`: Configuration files.
- `scripts/`: Helper scripts.

## Usage

```bash
go run main.go
```
