// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package publickey

import "github.com/toeirei/keymaster/client"

type createMsgImportResult struct {
	data      string
	algorithm string
	comment   string
}

type createMsgCreateResult struct {
	publicKeyId client.ID
	err         error
}

type CreateMsgCreated struct {
	PublicKeyId client.ID
}
