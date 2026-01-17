package core_test

import (
	"testing"

	core "github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/deploy"
	"github.com/toeirei/keymaster/internal/tui"
	"github.com/toeirei/keymaster/internal/ui"
)

func clearDefaults() {
	core.SetDefaultKeyReader(nil)
	core.SetDefaultKeyLister(nil)
	core.SetDefaultAccountSerialUpdater(nil)
	core.SetDefaultKeyImporter(nil)
	core.SetDefaultAuditWriter(nil)
	core.SetDefaultAccountManager(nil)
	core.SetDefaultDBInit(nil)
	core.SetDefaultDBIsInitialized(nil)
}

func checkAllSet(t *testing.T) {
	if core.DefaultKeyReader() == nil {
		t.Fatalf("DefaultKeyReader not set")
	}
	if core.DefaultKeyLister() == nil {
		t.Fatalf("DefaultKeyLister not set")
	}
	if core.DefaultAccountSerialUpdater() == nil {
		t.Fatalf("DefaultAccountSerialUpdater not set")
	}
	if core.DefaultKeyImporter() == nil {
		t.Fatalf("DefaultKeyImporter not set")
	}
	if core.DefaultAuditWriter() == nil {
		t.Fatalf("DefaultAuditWriter not set")
	}
	if core.DefaultAccountManager() == nil {
		t.Fatalf("DefaultAccountManager not set")
	}
}

func TestUIInitializeDefaults(t *testing.T) {
	clearDefaults()
	ui.InitializeDefaults()
	checkAllSet(t)
}

func TestTUIInitializeDefaults(t *testing.T) {
	clearDefaults()
	tui.InitializeDefaults()
	checkAllSet(t)
}

func TestDeployInitializeDefaults(t *testing.T) {
	clearDefaults()
	deploy.InitializeDefaults()
	checkAllSet(t)
}
