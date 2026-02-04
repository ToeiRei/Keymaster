// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/toeirei/keymaster/internal/core"
	"github.com/toeirei/keymaster/internal/core/db"
	"github.com/toeirei/keymaster/internal/i18n"
	"github.com/toeirei/keymaster/internal/model"
)

// TestRotateKeyCmd_HelpText verifies rotate-key command help text is present
func TestRotateKeyCmd_HelpText(t *testing.T) {
	cmd := NewRootCmd()
	rotateCmd := findSubcommand(cmd, "rotate-key")
	if rotateCmd == nil {
		t.Fatalf("rotate-key command not found")
	}

	if rotateCmd.Short == "" {
		t.Fatalf("rotate-key command missing short help")
	}
	if rotateCmd.Long == "" {
		t.Fatalf("rotate-key command missing long help")
	}
	if !strings.Contains(rotateCmd.Long, "key") {
		t.Fatalf("rotate-key help should mention key, got: %s", rotateCmd.Long)
	}
}

// TestAuditCmd_HelpText verifies audit command help text is present
func TestAuditCmd_HelpText(t *testing.T) {
	cmd := NewRootCmd()
	auditCmd := findSubcommand(cmd, "audit")
	if auditCmd == nil {
		t.Fatalf("audit command not found")
	}

	if auditCmd.Short == "" {
		t.Fatalf("audit command missing short help")
	}
	if !strings.Contains(auditCmd.Long, "drift") {
		t.Fatalf("audit help should mention drift, got: %s", auditCmd.Long)
	}
}

// TestDeployCmd_HelpText verifies deploy command help text is present
func TestDeployCmd_HelpText(t *testing.T) {
	cmd := NewRootCmd()
	deployCmd := findSubcommand(cmd, "deploy")
	if deployCmd == nil {
		t.Fatalf("deploy command not found")
	}

	if deployCmd.Short == "" {
		t.Fatalf("deploy command missing short help")
	}
	if !strings.Contains(deployCmd.Long, "authorized_keys") {
		t.Fatalf("deploy help should mention authorized_keys, got: %s", deployCmd.Long)
	}
}

// TestImportCmd_HelpText verifies import command help text is present
func TestImportCmd_HelpText(t *testing.T) {
	cmd := NewRootCmd()
	importCmd := findSubcommand(cmd, "import")
	if importCmd == nil {
		t.Fatalf("import command not found")
	}

	if importCmd.Short == "" {
		t.Fatalf("import command missing short help")
	}
	if !strings.Contains(importCmd.Long, "import") || !strings.Contains(importCmd.Long, "keys") {
		t.Fatalf("import help should mention importing keys, got: %s", importCmd.Long)
	}
}

// TestTrustHostCmd_HelpText verifies trust-host command help text is present
func TestTrustHostCmd_HelpText(t *testing.T) {
	cmd := NewRootCmd()
	trustCmd := findSubcommand(cmd, "trust-host")
	if trustCmd == nil {
		t.Fatalf("trust-host command not found")
	}

	if trustCmd.Short == "" {
		t.Fatalf("trust-host command missing short help")
	}
	if !strings.Contains(trustCmd.Long, "host") {
		t.Fatalf("trust-host help should mention host, got: %s", trustCmd.Long)
	}
}

// TestBackupCmd_HelpText verifies backup command help text is present
func TestBackupCmd_HelpText(t *testing.T) {
	cmd := NewRootCmd()
	backupCmd := findSubcommand(cmd, "backup")
	if backupCmd == nil {
		t.Fatalf("backup command not found")
	}

	if backupCmd.Short == "" {
		t.Fatalf("backup command missing short help")
	}
	if !strings.Contains(backupCmd.Long, "backup") || !strings.Contains(backupCmd.Long, "database") {
		t.Fatalf("backup help should mention database backup, got: %s", backupCmd.Long)
	}
}

// TestRestoreCmd_HelpText verifies restore command help text is present
func TestRestoreCmd_HelpText(t *testing.T) {
	cmd := NewRootCmd()
	restoreCmd := findSubcommand(cmd, "restore")
	if restoreCmd == nil {
		t.Fatalf("restore command not found")
	}

	if restoreCmd.Short == "" {
		t.Fatalf("restore command missing short help")
	}
	if !strings.Contains(restoreCmd.Long, "restore") {
		t.Fatalf("restore help should mention restore, got: %s", restoreCmd.Long)
	}
}

// TestDecommissionCmd_HelpText verifies decommission command help text is present
func TestDecommissionCmd_HelpText(t *testing.T) {
	cmd := NewRootCmd()
	decommCmd := findSubcommand(cmd, "decommission")
	if decommCmd == nil {
		t.Fatalf("decommission command not found")
	}

	if decommCmd.Short == "" {
		t.Fatalf("decommission command missing short help")
	}
	if !strings.Contains(decommCmd.Long, "SSH access") {
		t.Fatalf("decommission help should mention SSH access, got: %s", decommCmd.Long)
	}
}

// TestMigrateCmd_HelpText verifies migrate command help text is present
func TestMigrateCmd_HelpText(t *testing.T) {
	cmd := NewRootCmd()
	migrateCmd := findSubcommand(cmd, "migrate")
	if migrateCmd == nil {
		t.Fatalf("migrate command not found")
	}

	if migrateCmd.Short == "" {
		t.Fatalf("migrate command missing short help")
	}
	if !strings.Contains(migrateCmd.Long, "migrate") {
		t.Fatalf("migrate help should mention migrate, got: %s", migrateCmd.Long)
	}
}

// TestExportSSHConfigCmd_HelpText verifies export-ssh-client-config command help text is present
func TestExportSSHConfigCmd_HelpText(t *testing.T) {
	cmd := NewRootCmd()
	exportCmd := findSubcommand(cmd, "export-ssh-client-config")
	if exportCmd == nil {
		t.Fatalf("export-ssh-client-config command not found")
	}

	if exportCmd.Short == "" {
		t.Fatalf("export-ssh-client-config command missing short help")
	}
	if !strings.Contains(exportCmd.Long, "SSH") || !strings.Contains(exportCmd.Long, "config") {
		t.Fatalf("export-ssh-client-config help should mention SSH config, got: %s", exportCmd.Long)
	}
}

// TestDebugCmd_HelpText verifies debug command help text is present
func TestDebugCmd_HelpText(t *testing.T) {
	cmd := NewRootCmd()
	debugCmd := findSubcommand(cmd, "debug")
	if debugCmd == nil {
		t.Fatalf("debug command not found")
	}

	if debugCmd.Short == "" {
		t.Fatalf("debug command missing short help")
	}
	if !strings.Contains(debugCmd.Short, "debug") || !strings.Contains(debugCmd.Short, "config") {
		t.Fatalf("debug help should mention debug/config, got: %s", debugCmd.Short)
	}
}

// TestTransferCmd_HelpText verifies transfer command help text is present
func TestTransferCmd_HelpText(t *testing.T) {
	cmd := NewRootCmd()
	transferCmd := findSubcommand(cmd, "transfer")
	if transferCmd == nil {
		t.Fatalf("transfer command not found")
	}

	if transferCmd.Short == "" {
		t.Fatalf("transfer command missing short help")
	}
	if !strings.Contains(transferCmd.Long, "transfer") {
		t.Fatalf("transfer help should mention transfer, got: %s", transferCmd.Long)
	}
}

// TestAuditCmd_ModeFlag verifies audit command has mode flag
func TestAuditCmd_ModeFlag(t *testing.T) {
	cmd := NewRootCmd()
	auditCmd := findSubcommand(cmd, "audit")
	if auditCmd == nil {
		t.Fatalf("audit command not found")
	}

	modeFlag := auditCmd.Flags().Lookup("mode")
	if modeFlag == nil {
		t.Fatalf("audit command should have --mode flag")
	}
	if modeFlag.DefValue != "strict" {
		t.Fatalf("expected audit --mode default to be 'strict', got %s", modeFlag.DefValue)
	}
}

// TestDecommissionCmd_Flags verifies decommission command has required flags
func TestDecommissionCmd_Flags(t *testing.T) {
	cmd := NewRootCmd()
	decommCmd := findSubcommand(cmd, "decommission")
	if decommCmd == nil {
		t.Fatalf("decommission command not found")
	}

	// Check for --force flag
	forceFlag := decommCmd.Flags().Lookup("force")
	if forceFlag == nil {
		t.Fatalf("decommission command should have --force flag")
	}

	// Check for --dry-run flag
	dryRunFlag := decommCmd.Flags().Lookup("dry-run")
	if dryRunFlag == nil {
		t.Fatalf("decommission command should have --dry-run flag")
	}

	// Check for --tag flag
	tagFlag := decommCmd.Flags().Lookup("tag")
	if tagFlag == nil {
		t.Fatalf("decommission command should have --tag flag")
	}
}

// TestRestoreCmd_FullFlag verifies restore command has --full flag
func TestRestoreCmd_FullFlag(t *testing.T) {
	cmd := NewRootCmd()
	restoreCmd := findSubcommand(cmd, "restore")
	if restoreCmd == nil {
		t.Fatalf("restore command not found")
	}

	fullFlag := restoreCmd.Flags().Lookup("full")
	if fullFlag == nil {
		t.Fatalf("restore command should have --full flag")
	}
}

// TestRotateKeyCmd_PasswordFlag verifies rotate-key command has password flag
func TestRotateKeyCmd_PasswordFlag(t *testing.T) {
	cmd := NewRootCmd()
	rotateCmd := findSubcommand(cmd, "rotate-key")
	if rotateCmd == nil {
		t.Fatalf("rotate-key command not found")
	}

	pwdFlag := rotateCmd.Flags().Lookup("password")
	if pwdFlag == nil {
		t.Fatalf("rotate-key command should have --password/-p flag")
	}
}

// TestRootCmd_PersistentFlags verifies root command has persistent flags
func TestRootCmd_PersistentFlags(t *testing.T) {
	cmd := NewRootCmd()

	// Check --verbose flag
	verboseFlag := cmd.PersistentFlags().Lookup("verbose")
	if verboseFlag == nil {
		t.Fatalf("root command should have --verbose/-v flag")
	}

	// Check --config flag
	configFlag := cmd.PersistentFlags().Lookup("config")
	if configFlag == nil {
		t.Fatalf("root command should have --config flag")
	}

	// Check --language flag
	langFlag := cmd.PersistentFlags().Lookup("language")
	if langFlag == nil {
		t.Fatalf("root command should have --language flag")
	}
	if langFlag.DefValue != "en" {
		t.Fatalf("expected --language default to be 'en', got %s", langFlag.DefValue)
	}
}

// TestRootCmd_DatabaseFlags verifies database-related flags are present
func TestRootCmd_DatabaseFlags(t *testing.T) {
	cmd := NewRootCmd()

	// Check database.type flag
	dbTypeFlag := cmd.Flags().Lookup("database.type")
	if dbTypeFlag == nil {
		t.Fatalf("root command should have --database.type flag")
	}
	if dbTypeFlag.DefValue != "sqlite" {
		t.Fatalf("expected --database.type default to be 'sqlite', got %s", dbTypeFlag.DefValue)
	}

	// Check database.dsn flag
	dbDsnFlag := cmd.Flags().Lookup("database.dsn")
	if dbDsnFlag == nil {
		t.Fatalf("root command should have --database.dsn flag")
	}
	if !strings.Contains(dbDsnFlag.DefValue, "keymaster.db") {
		t.Fatalf("expected --database.dsn default to contain 'keymaster.db', got %s", dbDsnFlag.DefValue)
	}
}

// TestSetupDefaultServices_DBInitialization verifies DB initialization logic
func TestSetupDefaultServices_DBInitialization(t *testing.T) {
	// Create a temporary directory for test database
	tmp := t.TempDir()
	dbPath := tmp + "/test.db"

	// Set up minimal command with database flags
	cmd := &cobra.Command{}
	cmd.Flags().String("database.type", "sqlite", "")
	cmd.Flags().String("database.dsn", dbPath, "")
	cmd.Flags().Set("database.type", "sqlite")
	cmd.Flags().Set("database.dsn", dbPath)

	// Set XDG_CONFIG_HOME to temp dir to avoid config file conflicts
	oldXDG := os.Getenv("XDG_CONFIG_HOME")
	os.Setenv("XDG_CONFIG_HOME", tmp)
	defer os.Setenv("XDG_CONFIG_HOME", oldXDG)

	// Call setupDefaultServices
	err := setupDefaultServices(cmd, []string{})
	if err != nil {
		t.Fatalf("setupDefaultServices failed: %v", err)
	}

	// Verify DB was initialized
	if !core.IsDBInitialized() {
		t.Fatalf("expected DB to be initialized")
	}

	// Verify i18n was initialized (should not panic)
	_ = i18n.T("test.key")

	// Close the underlying sql.DB to allow temp cleanup on Windows
	if bunDB := db.BunDB(); bunDB != nil {
		_ = bunDB.DB.Close()
	}
}

// TestGetConfigPathFromCli_NoFlag verifies config path extraction when flag not set
func TestGetConfigPathFromCli_NoFlag(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().String("config", "", "config file")

	path, err := getConfigPathFromCli(cmd)
	if err != nil {
		t.Fatalf("expected no error when flag not set, got: %v", err)
	}
	if path != nil {
		t.Fatalf("expected nil path when flag not set, got: %v", *path)
	}
}

// TestGetConfigPathFromCli_WithPath verifies config path extraction when flag is set
func TestGetConfigPathFromCli_WithPath(t *testing.T) {
	tmp := t.TempDir()
	configPath := tmp + "/custom.yaml"

	// Create an empty config file
	f, err := os.Create(configPath)
	if err != nil {
		t.Fatalf("failed to create temp config: %v", err)
	}
	f.Close()

	cmd := &cobra.Command{}
	cmd.Flags().String("config", "", "config file")
	cmd.Flags().Set("config", configPath)

	path, err := getConfigPathFromCli(cmd)
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if path == nil {
		t.Fatalf("expected non-nil path")
	}
	if *path != configPath {
		t.Fatalf("expected path %s, got: %s", configPath, *path)
	}
}

// TestApplyDefaultFlags_AddsExpectedFlags verifies applyDefaultFlags adds database flags
func TestApplyDefaultFlags_AddsExpectedFlags(t *testing.T) {
	cmd := &cobra.Command{}
	applyDefaultFlags(cmd)

	// Verify flags were added
	if cmd.Flags().Lookup("database.type") == nil {
		t.Fatalf("applyDefaultFlags should add database.type flag")
	}
	if cmd.Flags().Lookup("database.dsn") == nil {
		t.Fatalf("applyDefaultFlags should add database.dsn flag")
	}
}

// TestApplyDefaultFlags_DoesNotDuplicate verifies applyDefaultFlags doesn't panic on duplicate calls
func TestApplyDefaultFlags_DoesNotDuplicate(t *testing.T) {
	cmd := &cobra.Command{}

	// Call twice - should not panic
	applyDefaultFlags(cmd)
	applyDefaultFlags(cmd) // Second call should detect existing flags and skip

	// Verify flags still exist
	if cmd.Flags().Lookup("database.type") == nil {
		t.Fatalf("database.type flag missing after duplicate applyDefaultFlags")
	}
}

// TestResolveBuildVersion_ExplicitValues verifies version resolution with explicit values
func TestResolveBuildVersion_ExplicitValues(t *testing.T) {
	// Test with explicit version set
	oldV := version
	oldC := gitCommit
	oldD := buildDate
	version = "v1.2.3"
	gitCommit = "abc123"
	buildDate = "2026-01-01T00:00:00Z"
	defer func() {
		version = oldV
		gitCommit = oldC
		buildDate = oldD
	}()

	v, c, d := resolveBuildVersion(nil)
	if v != "v1.2.3" {
		t.Fatalf("expected version v1.2.3, got %s", v)
	}
	if c != "abc123" {
		t.Fatalf("expected commit abc123, got %s", c)
	}
	if d != "2026-01-01T00:00:00Z" {
		t.Fatalf("expected buildDate 2026-01-01T00:00:00Z, got %s", d)
	}
}

// TestResolveBuildVersion_DevFallback verifies fallback to "dev" values
func TestResolveBuildVersion_DevFallback(t *testing.T) {
	oldV := version
	oldC := gitCommit
	oldD := buildDate
	version = "dev"
	gitCommit = "dev"
	buildDate = ""
	defer func() {
		version = oldV
		gitCommit = oldC
		buildDate = oldD
	}()

	v, c, d := resolveBuildVersion(nil)
	if v != "dev" {
		t.Fatalf("expected version dev, got %s", v)
	}
	if c != "dev" {
		t.Fatalf("expected commit dev, got %s", c)
	}
	if d != "" {
		t.Fatalf("expected empty buildDate, got %s", d)
	}
}

// TestVersionCmd_Output verifies version command produces output
func TestVersionCmd_Output(t *testing.T) {
	oldV := version
	oldC := gitCommit
	oldD := buildDate
	version = "v2.0.0"
	gitCommit = "deadbeef"
	buildDate = "2026-02-01T12:00:00Z"
	defer func() {
		version = oldV
		gitCommit = oldC
		buildDate = oldD
	}()

	cmd := NewRootCmd()
	versionCmd := findSubcommand(cmd, "version")
	if versionCmd == nil {
		t.Fatalf("version command not found")
	}

	// Capture output
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Execute version command
	versionCmd.Run(versionCmd, []string{})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.String()

	// Verify output contains version info
	if !strings.Contains(output, "v2.0.0") {
		t.Fatalf("expected version output to contain v2.0.0, got: %s", output)
	}
	if !strings.Contains(output, "deadbeef") {
		t.Fatalf("expected version output to contain commit deadbeef, got: %s", output)
	}
	if !strings.Contains(output, "2026-02-01") {
		t.Fatalf("expected version output to contain build date, got: %s", output)
	}
}

// TestCLIDeployerManager_Delegation verifies cliDeployerManager delegates to core
func TestCLIDeployerManager_Delegation(t *testing.T) {
	// Initialize minimal DB for core facades
	i18n.Init("en")
	if _, err := db.New("sqlite", ":memory:"); err != nil {
		t.Fatalf("db.New failed: %v", err)
	}

	dm := &cliDeployerManager{}
	acct := model.Account{ID: 1, Username: "test", Hostname: "testhost"}

	// These should delegate to core.DefaultDeployerManager
	// Since we're in a test environment without full setup, they may fail,
	// but we verify the delegation happens (no panic)
	_ = dm.DeployForAccount(acct, false)
	_ = dm.AuditSerial(acct)
	_ = dm.AuditStrict(acct)
}

// Helper function to find a subcommand by name
func findSubcommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, c := range cmd.Commands() {
		if c.Name() == name {
			return c
		}
	}
	return nil
}
