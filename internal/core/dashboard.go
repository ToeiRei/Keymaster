// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"sort"

	"github.com/toeirei/keymaster/internal/model"
)

// DashboardData holds aggregated values for the main dashboard.
type DashboardData struct {
	AccountCount       int
	ActiveAccountCount int
	PublicKeyCount     int
	GlobalKeyCount     int
	HostsUpToDate      int
	HostsOutdated      int
	AlgoCounts         map[string]int
	SystemKeySerial    int
	RecentLogs         []model.AuditLogEntry
}

// BuildDashboardData collects accounts, keys, system key and recent audit logs,
// and computes aggregated metrics for the dashboard.
// AccountReader exposes account read operations core needs for dashboard metrics.
type AccountReader interface {
	GetAllAccounts() ([]model.Account, error)
}

// KeyReader exposes key reads required by the dashboard.
type KeyReader interface {
	GetAllPublicKeys() ([]model.PublicKey, error)
	GetActiveSystemKey() (*model.SystemKey, error)
	GetSystemKeyBySerial(serial int) (*model.SystemKey, error)
}

// AuditReader exposes audit log reads required by the dashboard.
type AuditReader interface {
	GetAllAuditLogEntries() ([]model.AuditLogEntry, error)
}

// BuildDashboardData computes metrics using provided readers. Core no longer
// depends on DB packages directly; callers must supply implementations.
func BuildDashboardData(accounts AccountReader, keys KeyReader, audits AuditReader) (DashboardData, error) {
	var out DashboardData

	accs, err := accounts.GetAllAccounts()
	if err != nil {
		return out, err
	}

	klist, err := keys.GetAllPublicKeys()
	if err != nil {
		return out, err
	}

	sysKey, err := keys.GetActiveSystemKey()
	if err != nil {
		return out, err
	}

	logs, err := audits.GetAllAuditLogEntries()
	if err != nil {
		return out, err
	}

	out.AccountCount = len(accs)
	for _, acc := range accs {
		if acc.IsActive {
			out.ActiveAccountCount++
			if sysKey != nil && sysKey.Serial > 0 {
				if acc.Serial == sysKey.Serial {
					out.HostsUpToDate++
				} else {
					out.HostsOutdated++
				}
			}
		}
	}

	out.PublicKeyCount = len(klist)
	out.AlgoCounts = make(map[string]int)
	for _, k := range klist {
		if k.IsGlobal {
			out.GlobalKeyCount++
		}
		out.AlgoCounts[k.Algorithm]++
	}

	if sysKey != nil {
		out.SystemKeySerial = sysKey.Serial
	}

	const maxLogs = 5
	if len(logs) > maxLogs {
		out.RecentLogs = logs[:maxLogs]
	} else {
		out.RecentLogs = logs
	}

	// Ensure deterministic ordering keys where necessary by sorting the AlgoCounts keys
	// (callers can use the map; sorting is done when rendering).
	_ = sort.Ints

	return out, nil
}
