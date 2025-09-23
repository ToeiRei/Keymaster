# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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