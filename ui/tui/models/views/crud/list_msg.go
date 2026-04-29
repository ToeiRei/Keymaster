// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package crud

type listMsgReloaded[TDataGet any] struct {
	records []TDataGet
	err     error
}

type listMsgDeleteResult[TDataGet any] struct {
	record TDataGet
	err    error
}
