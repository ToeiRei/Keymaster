// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package bootstrap

import (
"testing"
"time"
)

func TestNewBootstrapSession(t *testing.T) {
s, err := NewBootstrapSession("alice", "example.com", "label", "tag")
if err != nil {
t.Fatalf("NewBootstrapSession returned error: %v", err)
}
if s == nil {
t.Fatal("expected session, got nil")
}
if s.ID == "" {
t.Error("expected non-empty session ID")
}
if s.TempKeyPair == nil {
t.Fatal("expected TempKeyPair to be set")
}
if len(s.TempKeyPair.GetPrivateKeyPEM()) == 0 {
t.Error("expected private key PEM to be non-empty")
}
if s.TempKeyPair.GetPublicKey() == "" {
t.Error("expected public key string to be non-empty")
}
if s.Status != StatusActive {
t.Errorf("expected status %s, got %s", StatusActive, s.Status)
}
if !s.CreatedAt.Before(s.ExpiresAt) {
t.Error("expected CreatedAt before ExpiresAt")
}
// ExpiresAt should be CreatedAt + BootstrapTimeout
if got := s.ExpiresAt.Sub(s.CreatedAt); got != BootstrapTimeout {
t.Errorf("unexpected timeout duration: got %v want %v", got, BootstrapTimeout)
}
}

func TestIsExpiredAndCleanup(t *testing.T) {
s, err := NewBootstrapSession("bob", "host.local", "lab", "")
if err != nil {
t.Fatalf("NewBootstrapSession returned error: %v", err)
}

// Not expired by default
if s.IsExpired() {
t.Error("expected session to be active (not expired)")
}

// Force expiration
s.ExpiresAt = time.Now().Add(-1 * time.Minute)
if !s.IsExpired() {
t.Error("expected session to be expired")
}

// Ensure Cleanup wipes sensitive data
priv := s.TempKeyPair.GetPrivateKeyPEM()
if len(priv) == 0 {
t.Fatal("expected private key before cleanup")
}
s.Cleanup()
if s.TempKeyPair != nil && len(s.TempKeyPair.GetPrivateKeyPEM()) != 0 {
t.Error("expected private key to be cleared after Cleanup")
}
}

func TestRegisterAndUnregisterSession(t *testing.T) {
// Reset registry state for test isolation
sessionsMutex.Lock()
activeSessions = make(map[string]*BootstrapSession)
sessionsMutex.Unlock()

s, err := NewBootstrapSession("carol", "host", "lbl", "")
if err != nil {
t.Fatalf("NewBootstrapSession returned error: %v", err)
}

RegisterSession(s)

sessionsMutex.RLock()
_, exists := activeSessions[s.ID]
sessionsMutex.RUnlock()
if !exists {
t.Fatal("expected session to be registered")
}

UnregisterSession(s.ID)

sessionsMutex.RLock()
_, exists = activeSessions[s.ID]
sessionsMutex.RUnlock()
if exists {
t.Fatal("expected session to be unregistered")
}
}

func TestInstallSignalHandlerIdempotent(t *testing.T) {
// Ensure calling InstallSignalHandler multiple times is safe
signalHandlerInstalled = false
InstallSignalHandler()
InstallSignalHandler()
if !signalHandlerInstalled {
t.Fatal("expected signal handler to be installed")
}
}
