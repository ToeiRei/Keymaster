// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"sort"

	"github.com/toeirei/keymaster/core/db"
	"github.com/toeirei/keymaster/core/model"
)

// DashboardData holds aggregated values for the main dashboard.
type DashboardData struct {
	AccountCount       int
	ActiveAccountCount int
	// PublicKeyCount     int
	// GlobalKeyCount     int
	// AlgoCounts         map[string]int
	HostsUpToDate   int
	HostsOutdated   int
	SystemKeySerial int
	RecentLogs      []model.AuditLogEntry
}

// BuildDashboardData collects accounts, keys, system key and recent audit logs,
// and computes aggregated metrics for the dashboard.
// AccountReader provides account read operations core needs for dashboard metrics.
type AccountReader interface {
	GetAllAccounts() ([]model.Account, error)
}

// AuditReader exposes audit log reads required by the dashboard.
type AuditReader interface {
	GetAllAuditLogEntries() ([]model.AuditLogEntry, error)
}

var _ AccountReader = (db.Store)(nil) // db.Store implements AccountReader
// var _ KeyReader = (db.Store)(nil)     // db.Store implements KeyReader
var _ AuditReader = (db.Store)(nil) // db.Store implements AuditReader

// BuildDashboardData computes metrics using provided readers. Core no longer
// depends on DB packages directly; callers must supply implementations.
func BuildDashboardData(store db.Store) (DashboardData, error) {
	var out DashboardData

	accs, err := store.GetAllAccounts()
	if err != nil {
		return out, err
	}

	// klist, err := store. GetAllPublicKeys()
	// if err != nil {
	// 	return out, err
	// }

	sysKey, err := store.GetActiveSystemKey()
	if err != nil {
		return out, err
	}

	logs, err := store.GetAllAuditLogEntries()
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

	// out.PublicKeyCount = len(klist)
	// out.AlgoCounts = make(map[string]int)
	// for _, k := range klist {
	// 	if k.IsGlobal {
	// 		out.GlobalKeyCount++
	// 	}
	// 	out.AlgoCounts[k.Algorithm]++
	// }

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
