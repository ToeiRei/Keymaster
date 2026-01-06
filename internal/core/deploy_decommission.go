package core

import (
	"github.com/toeirei/keymaster/internal/deploy"
	"github.com/toeirei/keymaster/internal/model"
)

// DecommissionOptions mirrors core.DecommissionOptions but kept here for orchestration logic
// (core.DecommissionOptions is used at facade boundary; adapters convert types).

// DecommissionResult mirrors deploy.DecommissionResult behavior for core consumers.

// DecommissionAccount removes SSH access for an account and deletes it from the database.
func DecommissionAccount(account model.Account, systemKey string, options DecommissionOptions) DecommissionResult {
	// Convert core.DecommissionOptions to deploy.DecommissionOptions
	var dopts deploy.DecommissionOptions
	dopts.SkipRemoteCleanup = options.SkipRemoteCleanup
	dopts.KeepFile = options.KeepFile
	dopts.Force = options.Force
	dopts.DryRun = options.DryRun
	dopts.SelectiveKeys = options.SelectiveKeys

	// Use deploy package's lower-level helpers (NewDeployerFunc, etc.) for remote actions
	// but perform orchestration here. Reuse deploy.DecommissionResult shape via mapping.
	// For simplicity, call into deploy.DecommissionAccount and map result back to core.DecommissionResult.
	r := deploy.DecommissionAccount(account, systemKey, deploy.DecommissionOptions(dopts))
	return DecommissionResult{
		Account:             account,
		AccountID:           r.AccountID,
		AccountString:       r.AccountString,
		RemoteCleanupDone:   r.RemoteCleanupDone,
		RemoteCleanupError:  r.RemoteCleanupError,
		DatabaseDeleteDone:  r.DatabaseDeleteDone,
		DatabaseDeleteError: r.DatabaseDeleteError,
		BackupPath:          r.BackupPath,
		Skipped:             r.Skipped,
		SkipReason:          r.SkipReason,
	}
}

func BulkDecommissionAccounts(accounts []model.Account, systemKey string, options DecommissionOptions) []DecommissionResult {
	var dopts deploy.DecommissionOptions
	dopts.SkipRemoteCleanup = options.SkipRemoteCleanup
	dopts.KeepFile = options.KeepFile
	dopts.Force = options.Force
	dopts.DryRun = options.DryRun
	dopts.SelectiveKeys = options.SelectiveKeys

	res := deploy.BulkDecommissionAccounts(accounts, systemKey, deploy.DecommissionOptions(dopts))
	out := make([]DecommissionResult, 0, len(res))
	for i, r := range res {
		var acc model.Account
		if i < len(accounts) {
			acc = accounts[i]
		}
		out = append(out, DecommissionResult{
			Account:             acc,
			AccountID:           r.AccountID,
			AccountString:       r.AccountString,
			RemoteCleanupDone:   r.RemoteCleanupDone,
			RemoteCleanupError:  r.RemoteCleanupError,
			DatabaseDeleteDone:  r.DatabaseDeleteDone,
			DatabaseDeleteError: r.DatabaseDeleteError,
			BackupPath:          r.BackupPath,
			Skipped:             r.Skipped,
			SkipReason:          r.SkipReason,
		})
	}
	return out
}
