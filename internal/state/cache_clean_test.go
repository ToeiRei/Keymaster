package state

import (
    "sync"
    "testing"
)

func TestPasswordMailbox_SetGetClear(t *testing.T) {
    PasswordCache.Clear()

    if got := PasswordCache.Get(); got != nil {
        t.Fatalf("expected nil on empty cache, got %v", got)
    }

    pass := []byte("s3cr3t")
    PasswordCache.Set(pass)

    got := PasswordCache.Get()
    if got == nil {
        t.Fatalf("expected value after Set, got nil")
    }
    if string(got) != string(pass) {
        t.Fatalf("expected %s, got %s", pass, got)
    }

    // Mutating returned slice shouldn't mutate internal value
    got[0] = 'X'
    got2 := PasswordCache.Get()
    if got2 == nil || got2[0] == 'X' {
        t.Fatalf("cache should return a copy; mutation leaked")
    }

    // Clear should wipe and subsequent Get returns nil
    PasswordCache.Clear()
    if got := PasswordCache.Get(); got != nil {
        t.Fatalf("expected nil after Clear, got %v", got)
    }
}

func TestPasswordMailbox_ConcurrentAccess(t *testing.T) {
    PasswordCache.Clear()
    defer PasswordCache.Clear()

    PasswordCache.Set([]byte("concurrent"))

    var wg sync.WaitGroup
    readers := 50
    wg.Add(readers)
    for i := 0; i < readers; i++ {
        go func() {
            defer wg.Done()
            for j := 0; j < 100; j++ {
                v := PasswordCache.Get()
                if v == nil {
                    t.Fatalf("expected non-nil during concurrent reads")
                }
            }
        }()
    }

    // Set a new value concurrently
    wg.Add(1)
    go func() {
        defer wg.Done()
        PasswordCache.Set([]byte("updated"))
    }()

    wg.Wait()
}
