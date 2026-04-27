// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package crud

type editMsgUpdateResult[TRecord any] struct {
	record TRecord
	err    error
}

type editMsgUpdated[TRecord any] struct {
	Record TRecord
}
