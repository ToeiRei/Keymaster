// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package core

import "strings"

// AuditActionRisk classifies an audit action into a risk category.
// Returns one of: "high", "medium", "low", "info".
func AuditActionRisk(action string) string {
	switch {
	case strings.HasPrefix(action, "DELETE_ACCOUNT"),
		strings.HasPrefix(action, "DELETE_PUBLIC_KEY"),
		strings.HasPrefix(action, "UNASSIGN_KEY"),
		strings.HasPrefix(action, "ROTATE_SYSTEM_KEY"),
		strings.HasPrefix(action, "BOOTSTRAP_FAILED"):
		return "high"
	case strings.HasPrefix(action, "TOGGLE_ACCOUNT_STATUS"),
		strings.HasPrefix(action, "TOGGLE_KEY_GLOBAL"),
		strings.HasPrefix(action, "UPDATE_ACCOUNT_LABEL"),
		strings.HasPrefix(action, "UPDATE_ACCOUNT_TAGS"),
		strings.HasPrefix(action, "ASSIGN_KEY"),
		strings.HasPrefix(action, "TRUST_HOST"),
		strings.HasPrefix(action, "CREATE_SYSTEM_KEY"):
		return "medium"
	case strings.HasPrefix(action, "ADD_ACCOUNT"),
		strings.HasPrefix(action, "ADD_PUBLIC_KEY"),
		strings.HasPrefix(action, "BOOTSTRAP_HOST"):
		return "low"
	default:
		return "info"
	}
}
