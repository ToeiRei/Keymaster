# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [UNRELEASED]

---

## [1.3.5] - 2025-09-29

### Added
- Add clipboard copy functionality for public keys (`c` in the public keys view).
- Add clipboard copy functionality to the deployment `authorized_keys` view.

### Fixed
- **Critical:** Fix a race condition during concurrent deployments where a `database is locked` error on SQLite could lead to a state inconsistency, effectively "losing" a host. A retry mechanism was added to the database update step to ensure successful deployments are correctly recorded.
- Fix migration format for `golang-migrate` compatibility.
- **Design fixes:** Long lists break the views

---

## [1.3.4] - 2025-09-26

### Added
- **Internationalization:** Completed translations for German, covering all CLI commands and TUI views.

### Changed
- **Database Migrations:** Refactored the database migration system to use separate SQL files for each supported database (SQLite, PostgreSQL, MySQL). This improves reliability and makes adding future schema changes easier.
- **CLI Initialization:** The root command initialization was refactored for better testability and to ensure consistent behavior.

### Fixed
- **Message Formatting:** Corrected several internationalization string formatting issues in the CLI and TUI to ensure messages display correctly.

---

## [1.3.3] - 2025-09-26

### Changed
- **Refactoring:** Consolidated duplicated `ed25519` key generation logic from the CLI and TUI into a single function in the `internal/crypto/ssh` package.
- **Database:** Enabled Write-Ahead Logging (WAL) mode for SQLite to improve concurrency and prevent `database is locked` errors.

### Fixed
- **Importer:** The key importer now correctly handles duplicate keys by checking for a specific database error (`db.ErrDuplicate`) instead of relying on string matching, which improves reliability across different database backends.

---

## [1.3.2] - 2025-09-25

### Changed
- **Internationalization:** Refactored the language switching logic to be more stable and prevent dynamic re-initialization.

### Fixed
- **SSH Agent Fallback:** Fixed a bug where key import or deployment would fail if a system key was present in the database but incorrect for the host, instead of correctly falling back to using the SSH agent.
- **TUI Navigation:** Restored `j/k` and `up/down` navigation in the "Deploy to Single Account" view when a filter is active.

---

## [1.3.1] - 2025-09-24
### Added
- **Project Governance:** Added standard open-source project files including `LICENSE` (MIT), `CODE_OF_CONDUCT.md`, and `CONTRIBUTING.md` to clarify contribution guidelines and project standards.
- **Code Documentation:** Added a lot of comments to clarify how things work.
### Fixed
- **Key Assignment:** Fixed a critical bug where assigning or unassigning a key to an account would fail due to swapped database parameters.

---

## [1.3.0] - 2025-09-23

### Added
- **Internationalization Support:** The TUI now supports multiple languages, with a language switcher and initial translations for German.

### Changed
- **UI Polish:** A comprehensive "Tender Loving Care" pass was applied to most views, including the dashboard, account/key management, deployment dialogs, and audit logs, to refine styling and improve user experience.
- **Key Assignment Rework:** The logic for assigning keys to accounts has been improved, especially regarding the handling of global keys.

### Fixed
- **Account Filter:** Resolved a UI glitch causing "jank" when filtering the accounts list.
- **Audit Log Layout:** Fixed a minor styling issue with the footer in the audit log view.

---

## [1.2.1] - 2024-09-24

### Changed
- **Audit Logic:** The `audit` command now performs a full content comparison of the remote `authorized_keys` file against the expected state, instead of only checking the serial number. This provides a much more accurate and reliable drift detection.
- **Import Command:** The `import` command now provides more detailed feedback, reporting errors for invalid key lines instead of skipping them silently.

### Fixed
- **GoReleaser Workflow:** Fixed multiple release failures by updating the workflow to be compatible with GoReleaser v2. This includes using a temporary file for release notes to prevent a "dirty" git workspace and using the correct action inputs.
- **Configuration Discovery:** Keymaster now prints a message when it automatically creates a default `.keymaster.yaml` file, improving user feedback on first run.
- **CLI Parsing:** Improved argument parsing in the `trust-host` command for consistency and robustness.
- **Deployment Compatibility:** The SFTP deployment logic now uses a backup-and-rename strategy, improving compatibility with SFTP servers that do not support atomic overwrites (e.g., on Windows).
- **Build Failures:** Resolved two separate build failures: one caused by a function being redeclared, and another by a package import conflict in `main.go`.
- **File Formatting:** Refined the `authorized_keys` file generator to ensure consistent formatting and a single trailing newline, adhering to POSIX standards.

### Security
- **Automatic System Key Hardening:** Keymaster now automatically prepends restrictive options (`command="internal-sftp"`, `no-port-forwarding`, etc.) to its system key during every deployment. This significantly hardens security by default, ensuring the system key can only be used for SFTP operations and not for interactive shells, even if compromised.

---

## [1.2.0] - 2024-09-23

This release introduces a massive overhaul of the user interface, migrating to a modern, responsive TUI.

### Added
- **Complete TUI Overhaul:** A brand new, modern interactive TUI built from the ground up with `lipgloss`.
- **Dashboard View:** The main menu now features a dashboard providing an at-a-glance overview of system status, key counts, and recent audit log activity.
- **Live Filtering:** All major lists (Accounts, Public Keys, Audit Log, Tags) now support live filtering. Simply press `/` to start searching.
- **Tag Autocompletion:** The "Add/Edit Account" form now provides autocomplete suggestions for tags based on existing tags in the database, reducing typos and improving consistency.

### Changed
- **Modernized UI Components:** All views, lists, and dialogs have been redesigned for a more consistent and professional look and feel.
- **Streamlined Workflows:**
  - The system key rotation flow is now a clean, modal-based confirmation directly from the main menu.
  - After adding or editing an account, it is now automatically selected in the list, allowing for immediate follow-up actions.
- **Improved Modals:** Confirmation dialogs for destructive actions are now graphical modals instead of simple text prompts.
- **Audit Log View:** The audit log is now a full-featured, filterable table with color-coded actions to quickly identify important events.

### Fixed
- Resolved numerous layout and alignment bugs across the application for a stable and pixel-perfect UI.
- Fixed a state synchronization bug where the account list would not refresh after an edit.
- Corrected list navigation and selection behavior to be consistent across all views.

## [1.1.0] - 2024-09-22

This release focused on adding powerful fleet management features and improving usability.

### Added
- **Tagging System:** Accounts can now be tagged with key-value pairs (e.g., `role:db`).
- **View by Tag:** A new TUI view to see all accounts grouped by their assigned tags.
- **Deploy to Tag:** New deployment option to push key updates to all accounts sharing a specific tag.
- **Global Public Keys:** Public keys can be marked as "global" to be automatically deployed to all managed accounts.
- **Remote Key Import:** Import public keys directly from a remote host's `authorized_keys` file via the TUI.
- **Account Labels:** Assign user-friendly labels to accounts for easier identification (e.g., `prod-web-01`).
- **Key Usage Reports:** View a report of all accounts a specific public key is assigned to.
- **Configuration File:** Keymaster now supports a `keymaster.yaml` configuration file for database settings.
- **Experimental DB Support:** Initial, experimental support for using PostgreSQL and MySQL as the database backend.
- **SSH Agent Integration:** Seamlessly uses your running SSH agent to bootstrap new hosts.

### Changed
- When a new account is added, Keymaster now automatically attempts to trust the host's public key.
- Improved resilience of SSH deployment logic.

## [1.0.0] - 2024-09-21

Initial public release of Keymaster.

### Added
- **Core Functionality:** Initial implementation of account and public key management.
- **SQLite Backend:** All data is stored in a local `keymaster.db` file.
- **System Key Management:** Generate and rotate the master system key used for deployments.
- **Single-Host & Fleet Deployment:** Deploy `authorized_keys` to individual accounts or the entire fleet.
- **Host Trusting:** A `trust-host` command and TUI flow to securely add new hosts by verifying their public key.
- **Drift Detection:** An `audit` command to check if hosts are in sync with the database.
- **CLI and TUI:** Both a scriptable command-line interface and a basic interactive TUI for all core operations.
- **Audit Logging:** All actions are logged to an audit trail in the database.
- **Account Activation Status:** Accounts can now be toggled between active and inactive states without being deleted.
- **Host Key Warnings:** The application now warns the user when trusting a host that uses a weak SSH key algorithm.