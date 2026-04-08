// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package form

const (
	ActionNone Action = iota
	ActionNext
	ActionPrev
	ActionSubmit
	ActionCancel
	ActionReset
)

type Action int
