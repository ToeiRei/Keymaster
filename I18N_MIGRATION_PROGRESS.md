# TUI i18n Migration — Progress & Handoff

Branch: `client-bun-rewrite`. This documents the migration of TUI texts, audit-log
actions and surfaced errors to i18n, so it can be continued from a fresh session.

## Goal

Move all user-facing TUI text, audit-log action codes, and commonly-surfaced errors
behind i18n, with translation resolved **as late as possible — at render time (`View`)
or the action that builds the text** — so the runtime language switch (Settings >
Language, already wired) reflects live without restart. Three key groups: TUI chrome
(bare per-view prefixes, no `tui.*` root), `audit.action.*`, and `errors.*`.

Languages: `en`, `de`, `art-x-ang` (Old English, a real full translation — match its
existing vocabulary; see `locales/active.art-x-ang.yaml`).

## Architecture / patterns established

- **`i18n.T(id, args...)`** (`ui/i18n/i18n.go`): legacy printf path (non-map varargs →
  `fmt.Sprintf`); **missing key → returns the id unchanged** (so literals pass through).
- **`i18n.Text` (`type Text string`)** (`ui/i18n/i18n.go`): a `fmt.Stringer` whose
  `String()` calls `T`. This is the *producing* side — pass `i18n.Text("some.key")`
  wherever a `fmt.Stringer` label is wanted; untranslated literals fall back to
  themselves. **Preferred over storing a string + resolving internally.**
- **`i18n.LocalizedError`** (`ui/i18n/error.go`): lazy error carrier holding
  `messageID + args (+ wrapped)`. `Error()` AND `String()` resolve via `T` at call time,
  so it's both an `error` and a `fmt.Stringer`. `NewError(key, args...)` /
  `WrapError(err, key, args...)` (the latter appends `err` as the trailing printf arg and
  keeps it for `Unwrap`). **Templates must use `%v`/`%s`, never `%w`** (Sprintf can't
  honor `%w`; the chain is preserved by `Unwrap`, not the string).
- **`i18n.TAuditAction(code)`** (`ui/i18n/error.go`): maps a stored audit action code
  (UPPER_SNAKE or dotted, incl. trailing `.requested`) to a localized label; humanizes
  unknown codes. Wired into `ui/tui/views/auditlog/model.go` (Action column `View`
  closure — render-time; stored codes unchanged).
- **Key-help freeze fix**: `ui/tui/util/keys/common_keys.go` builders resolve help text
  via `i18n.T("keys.*")`; every keymap's `ShortHelp/FullHelp` **rebuilds** its bindings
  from those builders (instead of returning init-frozen package-level fields). Because
  `keyhelp.View` calls `ShortHelp/FullHelp` each render, the footer follows the language.
  Stored keymap fields remain the source of truth for `key.Matches` (physical keys don't
  change with language). Same pattern for `Checkbox` toggle hint and `Button` click hint.
- **Locale parity test** (`locales/locales_test.go`): `managedPrefixes` lists the
  namespaces enforced to exist in all three files (and not equal their id). **Add every
  new fully-controlled namespace here.** The three files intentionally differ in legacy
  keys, so only managed prefixes are checked.

## Commits already landed (oldest → newest)

1. `ee004b6` feat(i18n): add lazy LocalizedError carrier and audit-action helper
2. `0fd3879` i18n(locales): add errors.* and audit.action.* namespaces
3. `2fbafbe` refactor(errors): route surfaced SSH/connector/client errors through i18n
4. `0f132b2` refactor(errors): migrate eager i18n error sites to lazy LocalizedError
5. `e1b97fe` feat(auditlog): translate audit action codes at render time
6. `d9dfddc` i18n(tui): resolve key-help descriptions at render time
7. `0382663` feat(tui): add Settings > Language selector with live language switch  **(authored by the user)** — introduced `i18n.Text` + migrated menu `Item.Name` to `fmt.Stringer`.
8. `1abfdd1` i18n(tui): resolve form-element labels, menu labels and header at render
9. `e7faf11` i18n(tui): translate CRUD framework chrome
10. `dbe2e8e` i18n(tui): Stringer labels + translate domain views, popups, deploy helper

(Note: two earlier commits had `@`-mangled messages from PowerShell here-string syntax
used in the Bash tool — fixed via a force-push with `--force-with-lease`. Always commit
multi-line messages with `git commit -F <file>` or a bash `<<'EOF'` heredoc, never
`@'...'@` in the Bash tool.)

## Namespaces added (all in en/de/art-x-ang, parity-guarded)

`errors.{ssh,connector,client}.*`, `audit.action.*`, `keys.*`, `crud.*`,
`account.*`, `public_key.*`, `link.*`, `popup.*`, `deploy.op_*`,
plus `menu.*` additions (public_keys/accounts/deploy/deploy_dirty/deploy_all/verify_all/
window_title). Domain views use **singular** prefixes (`account.`, `public_key.`,
`link.`) deliberately distinct from the legacy `accounts.*`/`account_form.*`/
`public_keys.*` keys (which describe the OLD UI and will be removed).

## IN PROGRESS — the tree does NOT currently build

The user is extending the `fmt.Stringer` convention **through the entire text-passing
layer** (deeper than commit 10 assumed). Uncommitted work-in-progress changes the
signatures of the popup/connector APIs to take `fmt.Stringer` instead of `string`:

- `messagepopup.Open(severity, message fmt.Stringer, cmd)`
- `choicepopup.Open(question fmt.Stringer, ...)` and `choicepopup.Choice.Name fmt.Stringer`
- `progresspopup` status/title → `fmt.Stringer` (`progresspopup/msg.go` touched)
- `connector.UserRequester.RequestChoice([]fmt.Stringer) int` (`connector/connector.go`)
- `LocalizedError` gained `String()` so it is a `fmt.Stringer` (an error can be passed
  directly to these APIs now).

Because commits 8–10 pass **plain strings** (`i18n.T(...)`, `fmt.Sprintf(i18n.T(...))`,
`err.Error()`) to these popups, those call sites now fail to compile. `go build ./...`
currently errors in `ui/tui/helpers/deploy`, `ui/tui/helpers/crud`,
`ui/tui/popups/selectpopup`, and their callers.

### The key design gap to resolve

`messagepopup`/`choicepopup` now want a `fmt.Stringer`, but many messages are
**parameterized** ("Error loading %s:\n%s" with entity name + error) — you can't pass a
pre-`Sprintf`'d string anymore. Options:

1. **Add a parameterized Stringer helper**, e.g. `i18n.Textf(key string, args ...any)
   fmt.Stringer` whose `String()` returns `T(key, args...)`. Then
   `messagepopup.Open(Error, i18n.Textf("crud.error_loading", entity, err), nil)`.
   `LocalizedError` already is exactly this shape (key+args+String) — a small `Textf`
   type generalizes it. **Recommended.**
2. Or a trivial literal Stringer for already-resolved/dynamic strings.

Pick one, then sweep every `messagepopup.Open`/`choicepopup.Open`/progress title/
`RequestChoice` call to pass a Stringer (often just the `LocalizedError` directly for
error paths, or `i18n.Text("key")` / `i18n.Textf("key", args...)`).

### Concrete unfinished call sites (from the last `go build ./...`)

- `ui/tui/helpers/deploy/operations.go`: `statusText` returns `string` but progress
  `Status` now wants `fmt.Stringer`; per-account result join; `choicepopup.Open(i18n.T(...))`;
  `NewButton(label ...)` label; `requester.go` `RequestChoice([]string)` vs `[]fmt.Stringer`.
- `ui/tui/helpers/deploy/{deploy,verify}.go`: `messagepopup.Open(Error, err.Error(), nil)`
  → pass the error (Stringer) directly, e.g. `messagepopup.Open(Error, <the error>, nil)`.
- `ui/tui/helpers/crud/{create,update,list}_model.go`: `fmt.Sprintf(i18n.T(...))` and
  `i18n.T("crud.close/reload")` passed to progress/message/choice popups and
  `choicepopup.Choice.Name` — convert to `i18n.Textf(...)` / `i18n.Text(...)`.
- `ui/tui/popups/selectpopup/model.go`: partially converted; `choicepopup.Choice.Name`
  now needs `i18n.Text`/`i18n.Textf`, and `error_loading_records` is parameterized.

## Remaining TODO (besides the Stringer sweep above)

- **`ui/tui/helpers/crud/new.go:~170`** — duplicate action still hardcodes
  `"Please select a "+singular+" to duplicate."`. Key `crud.select_to_duplicate`
  (`"Please select a %s to duplicate."`) is **already added** to all three locales;
  just wire it: `fmt.Sprintf(i18n.T("crud.select_to_duplicate"), ...)` (or `i18n.Textf`).
- **`selectpopup` `popup.loading_records`** — the `reload()` progress title
  ("Loading records") still literal; key `popup.loading_records` is already in locales.
- **Skipped intentionally** (not user-facing): `views/hostlist`, `views/hostedit`
  (panic stubs), `views/testview1`, `views/testpopup1`, `maintest`, and the test-popup
  scaffolding literals in `views/content/content.go` (`"Choose Account"`, `"You selected: "`,
  the test column titles). Leave as-is.
- **`core/db` `ErrDuplicate`** — left English on purpose (avoids adding an i18n import to
  `core/db` for a rarely-surfaced sentinel).
- **`client/testui/client.go`** — mock; its raw error strings were not converted (not prod).

## Gotchas / learnings

- **`%w` vs Sprintf**: any locale template rendered through `T`/`LocalizedError` must use
  `%v`/`%s`. Eight templates were flipped `%w`→`%v` in commit 4; if you add more lazy
  errors, flip theirs too.
- **SSH substring classifiers** (`core/deploy/ssh.go` `IsHostKeyError` etc.): host-key
  errors wrap dedicated sentinels (`errUnknownHostKey`, `errHostKeyMismatch`) and
  `IsHostKeyError` matches via `errors.Is` **plus** the original English substrings as a
  fallback (graceful degradation if the ssh lib flattens the chain). Don't remove the
  substring fallback.
- **Every new key must exist in all three locale files** — a missing key in de/art-x-ang
  renders the raw id (not the English fallback). The parity test catches this only for
  `managedPrefixes`.
- **Two parallel deploy trees**: `core/deploy_*.go` (flat, legacy `package core`) and
  `core/deploy/*.go` (new `package deploy`). Both were migrated.
- **Pre-existing test failures**: several `core/deploy` tests panic (nil DB in the bun
  layer on this environment) independent of this work — verified on HEAD. Not ours.
- **Commit messages**: use `git commit -F <file>` or bash `<<'EOF'`. The
  `Co-Authored-By` trailer uses `Claude Opus 4.8 (1M context)`.
- **User preference (saved to memory)**: positional/unkeyed struct literals for complete
  structs (compiler catches missing fields); keyed only for partial construction.

## Verify

- `go build ./...` && `go vet ./...` (ignore the pre-existing `client/testui` unkeyed-
  literal vet warnings).
- `go test ./locales/... ./ui/i18n/...` (parity + carrier tests).
- Run the TUI; toggle Settings > Language between en/de/art-x-ang and confirm menu,
  form labels, table headers, key-help footer, popups, header/dashboard all switch
  **without restart**. English leakage ⇒ a still-frozen construction-time site.
