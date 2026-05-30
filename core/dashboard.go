// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package core

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/toeirei/keymaster/core/model"
)

// DashboardData holds aggregated values for the main dashboard.
type DashboardData struct {
	AccountCount       int
	ActiveAccountCount int
	PublicKeyCount     int
	GlobalKeyCount     int
	AlgoCounts         map[string]int
	HostsUpToDate      int
	HostsOutdated      int
	SystemKeySerial    int
	RecentLogs         []model.AuditLogEntry
}

// BuildDashboardData collects accounts, keys, system key and recent audit logs,
// and computes aggregated metrics for the dashboard.

// DashboardReader provides minimal read operations needed for dashboard metrics.
type DashboardReader interface {
	GetAllAccounts() ([]model.Account, error)
	GetActiveSystemKey() (*model.SystemKey, error)
	GetAllAuditLogEntries() ([]model.AuditLogEntry, error)
}

type dashboardPublicKeyReader interface {
	GetAllPublicKeys() ([]model.PublicKey, error)
}

// BuildDashboardData computes metrics using provided reader. Core no longer
// depends on full DB packages directly; callers must supply a minimal DashboardReader.
func BuildDashboardData(reader DashboardReader) (DashboardData, error) {
	var out DashboardData

	accs, err := reader.GetAllAccounts()
	if err != nil {
		return out, err
	}

	sysKey, err := reader.GetActiveSystemKey()
	if err != nil {
		return out, err
	}

	logs, err := reader.GetAllAuditLogEntries()
	if err != nil {
		return out, err
	}

	accountsByID := make(map[int]model.Account, len(accs))
	for _, acc := range accs {
		accountsByID[acc.ID] = acc
	}

	keysByID := map[int]model.PublicKey{}
	if keyReader, ok := reader.(dashboardPublicKeyReader); ok {
		if keys, kerr := keyReader.GetAllPublicKeys(); kerr == nil {
			for _, k := range keys {
				keysByID[k.ID] = k
			}
			out.PublicKeyCount = len(keys)
			out.AlgoCounts = make(map[string]int)
			for _, k := range keys {
				if k.IsGlobal {
					out.GlobalKeyCount++
				}
				out.AlgoCounts[k.Algorithm]++
			}
		}
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

	if sysKey != nil {
		out.SystemKeySerial = sysKey.Serial
	}

	// Keep a larger tail in core so UIs can decide how many rows to render
	// based on available terminal real estate.
	const maxLogs = 25
	if len(logs) > maxLogs {
		out.RecentLogs = enrichDashboardLogs(logs[:maxLogs], accountsByID, keysByID)
	} else {
		out.RecentLogs = enrichDashboardLogs(logs, accountsByID, keysByID)
	}

	return out, nil
}

var (
	accountIDPattern = regexp.MustCompile(`(?i)(?:account\s*[:=]\s*|account_id\s*[:=]\s*|accountID\s*[:=]\s*)(\d+)`)
	keyIDPattern     = regexp.MustCompile(`(?i)(?:key\s*[:=]\s*|key_id\s*[:=]\s*|keyID\s*[:=]\s*)(\d+)`)
)

func enrichDashboardLogs(logs []model.AuditLogEntry, accountsByID map[int]model.Account, keysByID map[int]model.PublicKey) []model.AuditLogEntry {
	if len(logs) == 0 {
		return logs
	}

	out := make([]model.AuditLogEntry, len(logs))
	for i, logEntry := range logs {
		details := strings.TrimSpace(logEntry.Details)

		if accID, ok := extractID(accountIDPattern, details); ok {
			if acc, found := accountsByID[accID]; found {
				details = appendRef(details, fmt.Sprintf("account=%s(#%d)", acc.String(), accID))
			}
		}

		if keyID, ok := extractID(keyIDPattern, details); ok {
			if key, found := keysByID[keyID]; found {
				keyName := key.Comment
				if keyName == "" {
					keyName = fmt.Sprintf("%s:%s", key.Algorithm, truncateKeyData(key.KeyData, 10))
				}
				details = appendRef(details, fmt.Sprintf("key=%s(#%d)", keyName, keyID))
			}
		}

		logEntry.Details = details
		out[i] = logEntry
	}

	return out
}

func extractID(pattern *regexp.Regexp, input string) (int, bool) {
	m := pattern.FindStringSubmatch(input)
	if len(m) < 2 {
		return 0, false
	}
	id, err := strconv.Atoi(m[1])
	if err != nil {
		return 0, false
	}
	return id, true
}

func appendRef(details, ref string) string {
	if strings.Contains(details, ref) {
		return details
	}
	if details == "" {
		return ref
	}
	return details + " | " + ref
}

func truncateKeyData(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 3 {
		return string(r[:max])
	}
	return string(r[:max-3]) + "..."
}
