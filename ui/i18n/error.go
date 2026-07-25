// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package i18n

import "strings"

// LocalizedError is an error whose message is resolved through [T] lazily, at
// the moment Error is called. This keeps translation as late as possible: the
// error text follows the current language even if it was constructed under a
// different one.
//
// Because args holds a slice, LocalizedError is not comparable; construct it
// via [NewError]/[WrapError] (which return a pointer) and declare sentinels as
// pointers so errors.Is works by identity. Templates use fmt verbs (%v/%s), not
// %w; the wrapped chain is preserved by Unwrap, not by the rendered string.
type LocalizedError struct {
	messageID string
	args      []any
	wrapped   error
}

// Error implements the error interface, resolving the message in the current language.
func (e *LocalizedError) Error() string { return T(e.messageID, e.args...) }

// Error implements the fmt.Stringer interface, resolving the message in the current language.
func (e *LocalizedError) String() string { return T(e.messageID, e.args...) }

// Unwrap returns the wrapped error so errors.Is/As traverse the chain.
func (e *LocalizedError) Unwrap() error { return e.wrapped }

// MessageID returns the i18n key backing this error, for tests and introspection.
func (e *LocalizedError) MessageID() string { return e.messageID }

// NewError creates a LocalizedError for messageID with optional printf args.
func NewError(messageID string, args ...any) *LocalizedError {
	return &LocalizedError{messageID: messageID, args: args}
}

// WrapError creates a LocalizedError that wraps err. err is both retained for
// Unwrap and appended as the final printf arg, so a trailing %v in the template
// renders its message (mirroring fmt.Errorf("...: %w", args..., err)).
func WrapError(err error, messageID string, args ...any) *LocalizedError {
	return &LocalizedError{messageID: messageID, args: append(args, err), wrapped: err}
}

// TAuditAction resolves a stored audit action code to a human-readable label in
// the current language. Codes come in two vocabularies: UPPER_SNAKE_CASE
// (e.g. "ADD_ACCOUNT") and dotted lower-case (e.g. "public_key.create"), and a
// code may carry a trailing ".requested" suffix. Stored codes are never changed;
// this is display-only. Unknown codes fall back to a humanized form of the code.
func TAuditAction(code string) string {
	code = strings.TrimSpace(code)
	if code == "" {
		return "-"
	}

	requested := strings.HasSuffix(code, ".requested")
	if requested {
		code = strings.TrimSuffix(code, ".requested")
	}

	// Uppercase codes normalize to lower-case; dotted codes pass through.
	normalized := strings.ToLower(code)
	key := "audit.action." + normalized

	label := T(key)
	if label == key {
		// Missing key: humanize the raw code.
		label = humanizeActionCode(code)
	}

	if requested {
		label = T("audit.action.requested_suffix", label)
	}
	return label
}

// humanizeActionCode turns "add_account" or "public_key.create" into
// "Add Account" / "Public Key Create".
func humanizeActionCode(code string) string {
	fields := strings.FieldsFunc(code, func(r rune) bool { return r == '_' || r == '.' })
	for i, f := range fields {
		if f == "" {
			continue
		}
		fields[i] = strings.ToUpper(f[:1]) + strings.ToLower(f[1:])
	}
	return strings.Join(fields, " ")
}
