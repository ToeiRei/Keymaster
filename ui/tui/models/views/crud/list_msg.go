// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package crud

type listMsgReloaded[TRecord any] struct {
	records []TRecord
	err     error
}

type listMsgDeleteResult[TRecord any] struct {
	record TRecord
	err    error
}
