// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// package state provides a secure, in-memory cache for transient application
// state, such as passwords or passphrases, that need to be shared between
// different parts of the application (e.g., CLI flags and TUI components).
package state

import "sync"

// PasswordCache is a simple, concurrency-safe, in-memory "mailbox" for
// temporarily storing a password or passphrase. It uses a byte slice instead of
// a string so that the sensitive data can be explicitly zeroed out after use.
var PasswordCache = &passwordMailbox{
	// value is initialized to nil
}

type passwordMailbox struct {
	value []byte
	mu    sync.RWMutex
}

// Set stores a copy of the password in the cache. It overwrites any existing value.
func (p *passwordMailbox) Set(pass []byte) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if pass == nil {
		p.value = nil
		return
	}
	// Store a copy so the caller's original slice isn't held by the cache.
	p.value = make([]byte, len(pass))
	copy(p.value, pass)
}

// Get retrieves a copy of the password from the cache.
// The caller is responsible for zeroing out the returned byte slice after use.
// This method is safe for concurrent use by multiple goroutines.
func (p *passwordMailbox) Get() []byte {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.value == nil {
		return nil
	}

	// Return a copy so that multiple goroutines can get the password
	// and one wiping its copy doesn't affect others.
	passCopy := make([]byte, len(p.value))
	copy(passCopy, p.value)
	return passCopy
}

// Clear securely wipes the password from the cache memory.
func (p *passwordMailbox) Clear() {
	p.mu.Lock()
	defer p.mu.Unlock()
	for i := range p.value {
		p.value[i] = 0
	}
	p.value = nil
}
