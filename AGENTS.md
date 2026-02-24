# Faliactl Architecture Knowledge Base

This project is built using Go, heavily leveraging the `charm.land` ecosystem to provide a beautiful CLI/TUI experience for interacting with Ostfalia university data.

## Project Structure

- `cmd/`: Contains the Cobra CLI entrypoints.
  - `root.go`: Root command definition.
  - `export.go`: Exposes the non-interactive scraper export functionality via flags (`--group`, `--output`).
  - `mensa.go`: Exposes the cafeteria fetching API via flags (`--campus`, `--date`).
  - `transit.go`: Exposes the HAFAS transit routing and upcoming weekly ICS export.
  - `config.go`: Allows CLI configuration of user preferences (like home base).
  - `interactive.go`: Launches the main Charm Huh based TUI menu.
- `pkg/scraper/`: The backend module responsible for fetching and parsing HTML data from the university servers.
  - Features `client.go` to handle HTTP connections.
  - Uses `goquery` to parse `schedule.html` (for group lists) and specific schedule IDs (e.g. `161902.html`) for course data.
  - Includes `integration_test.go` to actively verify the university intranet layouts haven't broken.
- `pkg/exporter/`: Takes the parsed schedule blocks and maps them into an iCalendar format using `golang-ical`.
- `pkg/mensa/`: The backend API module responsible for pulling live JSON from `api.stw-on.de`.
  - Includes models mapping pricing tiers, food variants (vegan), and allergen metadata.
  - Includes `integration_test.go` directly hitting the remote API to ensure schemas are stable.
- `pkg/transit/`: The backend transit routing engine wrapping `v6.db.transport.rest`.
  - Implements a resilient `Client` with a retry-loop and custom User-Agent to handle 503 public tracking limits.
  - Contains `integration_test.go` parsing dynamic HAFAS structures.
- `pkg/config/`: A simple OS-agnostic JSON storage module designed to remember user variables like `home_address`, `accent_color`, and `saved_courses` in a local dotfile (`~/.faliactl.json`). This module allows `faliactl` to instantly bypass interactive selection menus when default settings are populated.
- `pkg/scraper/cache.go`: Implements a 12-hour local caching system mapping API responses to `~/.faliactl_cache`. This strictly mitigates the aggressive load times from hitting the Intranet on repetitive daily commands like `Weekly Commute Planner`.
- `pkg/tui/`: The UI components built using `huh.Form`. Provides fuzzy-searchable multi-select lists for schedules, cafeterias, and transit.
  - Implements a dynamic styling builder via `GetTheme()` in `app.go`. This securely decrypts the user's saved hex color preference and dynamically re-binds Lipgloss variables across all TUI screens.

## Architectural Philosophy
Any future extensions or feature work on `faliactl` must adhere to these three core design pillars:
1. **Offline-First & Fast**: The university intranet is historically slow. `faliactl` must heavily leverage `pkg/config/` for persistent preferences and `pkg/scraper/cache.go` to aggressively cache HTTP responses, ensuring CLI commands execute in under 1 second whenever possible.
2. **Unix Philosophy / Pipeline Ready**: The CLI commands in `cmd/` (like `export` and `transit`) must always support raw stdout piping to allow users to build shell scripts around them. Complex TUI elements (`huh.Form`) should remain strictly cordoned off inside `pkg/tui/` and `interactive.go`.
3. **Strict Separation of Concerns**: Models and API logic (`pkg/scraper`, `pkg/mensa`, `pkg/transit`) must never import or be aware of UI frameworks (`charmbracelet/huh` or `lipgloss`). This enforces a clean MVC pattern where the terminal UI is merely a presentation layer wrapped around highly-testable core business logic.
4. **Dependency Minimalism**: `faliactl` intentionally avoids massive web frameworks or heavy ORMs. If a feature can be built utilizing standard Go libraries (e.g., `net/http` and `encoding/json`), do not add an external GitHub package to `go.mod`.

## Future Extensibility
- **Adding new commands**: The application uses Cobra, so adding a new sub-command is as easy as creating a new file in `cmd/` and adding it to the `rootCmd`.
- **Testing Philosophy**: `faliactl` relies heavily on two layers of testing:
  1. **Live Integration Tests**: Tests ending in `_test.go` that ping remote endpoints to ensure the backend wrappers remain functionally valid over time. Always include a live integration test when implementing a new external data source.
  2. **Mocked Unit Tests**: Robust unit tests parsing injected simulated JSON objects (like `httptest.Server`) to guarantee the offline extraction logic does not panic if upstream responses omit fields or throw HTTP 503s.

## Agent Instructions
- **Self-Updating Knowledge Base**: As an AI agent working on this repository, you must ALWAYS proactively update this `agents.md` file and the `README.md` file to reflect any new modules, structural changes, design patterns, or CLI commands that you implement over future conversations. Maintaining this documentation as the single source of truth is a strict requirement for all codebase modifications.
- **Git Operations**: NEVER `git commit` or `git push` autonomously without explicit permission from the user. You must always ask for approval before performing any git operations that modify the repository history or remote state.
- **Code Quality & Formatting**: Before asking the user to commit, you must ensure that `go fmt ./...` and `go vet ./...` have been executed and return zero errors. This repository maintains a strictly clean state.
- **Cross-Platform Compatibility**: `faliactl` is an OS-agnostic CLI. You must NEVER hardcode UNIX-style paths (e.g., `~/.faliactl.json`). Always use `os.UserHomeDir()` and `filepath.Join()` to ensure the CLI does not crash when compiled for Windows users.
