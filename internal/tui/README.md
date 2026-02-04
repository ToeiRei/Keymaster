# Legacy TUI (Deprecated)

> ⚠️ **DEPRECATED**: This TUI implementation is legacy code maintained for backward compatibility only.

## Status

This package is **deprecated** and kept together with duct tape, hopes, and prayers. It is excluded from:
- Test coverage metrics
- Code quality gates
- Refactoring efforts

## Maintenance Policy

- **No new features**: Do not add new functionality to this package
- **Minimal fixes only**: Only critical bug fixes should be applied
- **Migration path**: New UI development should use the `ui/tui` package instead

## Why Deprecated?

The original TUI was built as a proof-of-concept and has accumulated technical debt. A new, cleaner TUI implementation is being developed in `ui/tui` with:
- Better architecture
- Proper separation of concerns
- Modern Bubble Tea patterns
- Full test coverage

## For Developers

If you're working on UI features:
- Use `ui/tui` for new TUI code
- Use `ui/cli` for CLI commands
- Avoid touching this package unless absolutely necessary

## Removal Timeline

This package will be removed in a future major version once the new TUI implementation reaches feature parity.
