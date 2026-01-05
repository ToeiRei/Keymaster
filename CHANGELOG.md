# **Changelog**

_All notable changes to Keymaster are documented here. This project follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/) and adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html)._

---

## **[Unreleased] – Future Release**

This release brings major improvements to reliability, performance, and day‑to‑day usability across Keymaster. It includes expanded test coverage, internal stability work, and several new features that make managing SSH keys and deployments smoother.

### Added

- **Export `authorized_keys` to disk**  
  Allows offline or air‑gapped updates using local files.
- **Public‑key expiration support**  
  Keys can now expire, be filtered by status, and be reactivated or deactivated.
- **Improved search and filtering**  
  Faster, more accurate matching with better tag suggestions.
- **Deployment planning**  
  Keymaster now builds a structured deployment plan before applying changes.

### Changed

- **Faster TUI performance**  
  Noticeably improved filtering and autocompletion responsiveness.
- **More consistent database behavior**  
  Unified handling across SQLite, PostgreSQL, and MySQL.
- **Improved CI reliability**  
  Broader test coverage and more stable test environments.
- **Cleaner internal architecture**  
  Deterministic logic consolidated into a shared core for long‑term maintainability.

### Fixed

- **Global key indicator restored**  
  Global keys now display correctly in the UI.
- **Bootstrap command input**  
  The command field now renders at the correct height.
- **More reliable database tests**  
  Eliminated several intermittent failures in CI.

---

## **[1.5.1] – 2025‑12‑30**

A patch release focused on stability, diagnostics, and small workflow improvements.

### Added

- **`debug` command** to inspect runtime configuration, flags, and environment.

### Changed

- Improved configuration loading and default config generation.
- CI now enforces formatting and vetting checks.

### Fixed

- Clearer error messages for broken configs.
- Test fixes for SSH key parsing.
- `.gitignore` cleanup to avoid hiding the main binary in development.

---

## **[1.5.0] – 2025‑11‑20**

A major update to the data layer and build pipeline.

### Added

- **Build metadata** (commit SHA and build date) embedded in the binary.
- **`version` command** to display build information.
- **Comprehensive database tests** for the new Bun‑based data layer.
- **Automated CI/CD pipeline** for testing and building.

### Changed

- **Database layer migrated to Bun**, improving consistency and type safety.
- Updated cryptography and other dependencies.

### Fixed

- Duplicate CLI flag definitions.
- GitHub Actions permission issues.

---

## **[1.4.3] – 2025‑10‑14**

Focused on encrypted key handling and TUI workflow improvements.

### Added

- **Interactive passphrase prompts** for encrypted system keys.
- **Tag autocompletion** in the account editor.

### Changed

- Improved SSH authentication fallback behavior.

### Fixed

- Account editing issues.
- Several TUI navigation and state bugs.
- Incorrect status messages after deployments.

---

## **[1.4.0] – 2025‑10‑01**

A major feature release introducing database portability, a more resilient bootstrap process, and dashboard enhancements.

### Added

- **Backup, restore, and migrate commands** for full database portability.
- **Resilient bootstrap workflow** with temporary key cleanup and crash recovery.
- **Automatic cleanup of expired bootstrap sessions**.
- **Decommission command** to safely remove accounts.
- **Dashboard improvements** showing deployment status and key type breakdowns.

### Changed

- Configuration files now follow platform‑specific standards.
- More robust host parsing.
- Completed German translations.

### Fixed

- Config loading issues.
- TUI window size persistence.

### Security

- Hardened bootstrap cleanup to prevent key replacement on untrusted hosts.

---

## **[1.3.5] – 2025‑09‑28**

### Added

- Clipboard copy functionality for public keys and deployment views.

### Fixed

- Critical SQLite race condition during concurrent deployments.
- Migration format issues.
- Layout issues with long lists.

---

## **[1.3.4] – 2025‑09‑26**

### Added

- Completed German translations for all CLI and TUI views.

### Changed

- Database migrations reorganized for reliability.
- CLI initialization improved for consistency.

### Fixed

- Internationalization formatting issues.

---

## **[1.3.3] – 2025‑09‑24**

### Changed

- Consolidated key generation logic.
- Enabled WAL mode for SQLite to improve concurrency.

### Fixed

- More reliable duplicate‑key detection during import.

---

## **[1.3.2] – 2025‑09‑24**

### Changed

- More stable language switching logic.

### Fixed

- SSH agent fallback issues.
- TUI navigation in filtered views.

---

## **[1.3.1] – 2025‑09‑24**

### Added

- Standard open‑source governance files.
- Additional code documentation.

### Fixed

- Key assignment parameter bug.

---

## **[1.3.0] – 2025‑09‑23**

A major UI and UX overhaul.

### Added

- Full TUI redesign with modern components.
- Dashboard view.
- Live filtering across all major lists.
- Tag autocompletion.

### Changed

- Polished UI across all views.
- Streamlined workflows for key rotation and account editing.
- Improved confirmation dialogs.

### Fixed

- Numerous layout issues.
- State synchronization bugs.
- Navigation inconsistencies.

---

## **[1.2.1] – 2025‑09‑23**

### Changed

- Audit command now performs full content comparison.
- Import command provides clearer feedback.

### Fixed

- GoReleaser workflow issues.
- Config discovery messaging.
- CLI parsing improvements.
- SFTP deployment compatibility.
- Build failures.
- `authorized_keys` formatting consistency.

### Security

- Automatic hardening of system keys during deployment.

---

## **[1.2.0] – 2025‑09‑23**

A complete TUI overhaul and major usability improvements.

### Added

- Modern TUI built with lipgloss.
- Dashboard view.
- Live filtering.
- Tag autocompletion.

### Changed

- Modernized UI components.
- Streamlined workflows.
- Improved modals and audit log view.

### Fixed

- Numerous layout and state issues.

---

## **[1.1.0] – 2025‑09‑22**

### Added

- Tagging system.
- View by tag.
- Deploy to tag.
- Global public keys.
- Remote key import.
- Account labels.
- Key usage reports.
- Configuration file support.
- Experimental PostgreSQL/MySQL support.
- SSH agent integration.

### Changed

- Automatic host trust on account creation.
- Improved SSH deployment resilience.

---

## **[1.0.0] – 2025‑09‑21**

Initial public release.

### Added

- Account and key management.
- SQLite backend.
- System key rotation.
- Single‑host and fleet deployments.
- Host trust workflow.
- Drift detection.
- CLI and TUI interfaces.
- Audit logging.
- Account activation toggles.
- Weak key warnings.
