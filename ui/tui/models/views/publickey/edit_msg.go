// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package publickey

import "github.com/toeirei/keymaster/client"

type editMsgLoadResult struct {
	publicKey client.PublicKey
	err       error
}

type editMsgUpdateResult struct {
	err error
}

type EditMsgUpdated struct {
	PublicKeyId client.ID
}
