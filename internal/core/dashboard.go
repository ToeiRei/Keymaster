// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

package core

import (
	"sort"

	"github.com/toeirei/keymaster/internal/db"
	"github.com/toeirei/keymaster/internal/model"
	"github.com/toeirei/keymaster/internal/ui"
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
func BuildDashboardData() (DashboardData, error) {
	var out DashboardData

	accounts, err := db.GetAllAccounts()
	if err != nil {
		return out, err
	}

	km := ui.DefaultKeyManager()
	if km == nil {
		return out, nil
	}
	keys, err := km.GetAllPublicKeys()
	if err != nil {
		return out, err
	}

	sysKey, err := db.GetActiveSystemKey()
	if err != nil {
		return out, err
	}

	logs, err := db.GetAllAuditLogEntries()
	if err != nil {
		return out, err
	}

	out.AccountCount = len(accounts)
	for _, acc := range accounts {
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

	out.PublicKeyCount = len(keys)
	out.AlgoCounts = make(map[string]int)
	for _, k := range keys {
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
