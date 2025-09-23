# Contributing to Keymaster

First off, thank you for considering contributing to Keymaster! It’s people like you that make open source such a great community. We're excited to see what you bring to the project.

By contributing, you agree that your contributions will be licensed under the project's MIT License.


This document outlines the development process and conventions we follow. Please help us keep the codebase clean, maintainable, and easy to navigate for everyone.

**Before submitting changes, always test your work manually to ensure it behaves as expected.**

## Getting Started

1.  **Fork & Clone**: Fork the repository on GitHub and clone your fork locally.
2.  **Build**: The project uses standard Go modules. Build with:
    ```sh
    go build ./cmd/keymaster
    ```
3.  **Run**: Launch the TUI with:
    ```sh
    keymaster
    ```
    Or run directly:
    ```sh
    go run ./cmd/keymaster
    ```

## How to Contribute

The typical workflow is:

1.  Create a new branch for your feature or bugfix from the `main` branch. Please use a descriptive branch name (e.g., `feat/add-new-widget`, `fix/login-bug`).
2.  Make your changes. Adhere to the Coding Guidelines below.
3.  Test your changes to confirm they work as intended.
4.  Ensure your code is well-documented.
5.  Commit your changes using the Commit Message Convention.
6.  Push your branch to your fork and open a Pull Request to the main Keymaster repository.

## Coding Guidelines

We've established a consistent style throughout the project. Sticking to it makes the whole codebase feel cohesive.

### Go Formatting

All Go code must be formatted with `gofmt`. Most editors can be configured to do this automatically on save.

### Documentation

This is a big one for us! We take pride in having a well-documented codebase.

*   **Every file** must have a file-level comment at the top explaining its purpose.
*   **Every exported function, type, and variable** must have a GoDoc comment.
*   **Most non-trivial unexported functions** should also have a comment explaining their purpose.
*   Comments should start with the name of the symbol they are documenting (e.g., `// MyFunction does...`).
*   Use the `// import "..."` comment for package documentation.

### Terminal User Interface (TUI)

The TUI is built with the Bubble Tea framework, following the Model-View-Update (MVU) architecture.

*   **State Management**: Each major view (e.g., `accounts`, `public_keys`) has its own model (`accountsModel`, `publicKeysModel`) and manages its own state. The `mainModel` in `tui.go` acts as a router.
*   **Styling**: All UI styles are defined in `internal/tui/styles.go`. Please use these shared styles to maintain a consistent look and feel. Do not define inline styles.
*   **Internationalization (i18n)**: All user-facing strings **must** use the i18n system.
    *   Use `i18n.T("message.id")` to get a translated string.
    *   Add new string IDs to `internal/i18n/locales/en.yaml`. If you speak another language, feel free to add it to the corresponding file!

### Database

All database interactions are abstracted through the `Store` interface in `internal/db/store.go`.

*   Never interact with the `*sql.DB` object directly from the application logic (e.g., from `cmd` or `tui`).
*   All database calls should go through the functions in the `internal/db` package (e.g., `db.GetAllAccounts()`).
*   If you need to change the database schema, you must update the migration functions (`runMigrations`, `runPostgresMigrations`, etc.) in the respective driver files (`sqlite.go`, `postgres.go`).

### Error Handling

*   In command-line functions (`cobra.Command.Run`), return errors instead of calling `log.Fatalf()`. This allows Cobra to handle error printing.
*   In the TUI, display errors gracefully to the user within the UI. Don't `panic` or `os.Exit`.

### Security Practices

*   Never store user private keys in the database.
*   Treat `keymaster.db` as a secret (restrict permissions).
*   All system keys deployed must be SFTP-only.

---

If you have questions or want to propose a feature, please open an issue or start a discussion.

Thank you again for your interest in contributing! We look forward to your pull requests.