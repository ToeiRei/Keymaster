// Copyright (c) 2025 ToeiRei
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.

// i18n-linter is a tool to check for missing or orphaned translation keys.
// It scans the Go source code for i18n.T() calls and compares them against
// the YAML locale files to ensure consistency.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Location stores the file and line number of a found string.
type Location struct {
	Filepath string
	Line     int
}

const (
	localesDir    = "internal/i18n/locales"
	primaryLocale = "active.en.yaml"
	projectRoot   = "."
)

func main() {
	fmt.Println("🔍 Running i18n linter...")

	// 1. Find all keys used in the Go source code.
	usedKeys, err := findUsedKeys(projectRoot)
	if err != nil {
		fmt.Printf("❌ Error finding used keys: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Found %d unique translation keys used in source code.\n", len(usedKeys))

	// 2. Load all locale files.
	localeFiles, err := filepath.Glob(filepath.Join(localesDir, "*.yaml"))
	if err != nil {
		fmt.Printf("❌ Error finding locale files: %v\n", err)
		os.Exit(1)
	}

	// 3. Load the primary locale as the source of truth.
	primaryKeys, err := loadKeysFromLocale(filepath.Join(localesDir, primaryLocale))
	if err != nil {
		fmt.Printf("❌ Error loading primary locale '%s': %v\n", primaryLocale, err)
		os.Exit(1)
	}
	fmt.Printf("✅ Loaded %d keys from primary locale (%s).\n\n", len(primaryKeys), primaryLocale)

	// 4. Find potentially untranslated strings in the Go source code.
	untranslatedStrings, err := findUntranslatedStrings(projectRoot, usedKeys, primaryKeys)
	if err != nil {
		fmt.Printf("❌ Error finding untranslated strings: %v\n", err)
		os.Exit(1)
	}

	hasMissingKeys := false
	hasOrphanedKeys := false

	// 4. Check for orphaned keys in the primary locale.
	fmt.Println("--- Checking for Orphaned Keys (in primary locale but not used in code) ---")
	orphanedFound := false
	var orphanedKeys []string
	for key := range primaryKeys {
		if _, exists := usedKeys[key]; !exists {
			orphanedKeys = append(orphanedKeys, key)
		}
	}
	sort.Strings(orphanedKeys)
	for _, key := range orphanedKeys {
		fmt.Printf("  - Orphaned: %s\n", key)
		orphanedFound = true
		hasOrphanedKeys = true
	}
	if !orphanedFound {
		fmt.Println("  ✨ None found.")
	}
	fmt.Println()

	// 5. Check other locales for missing keys.
	fmt.Println("--- Checking for Missing Keys (in primary locale but not in others) ---")
	for _, file := range localeFiles {
		if filepath.Base(file) == primaryLocale {
			continue
		}

		fmt.Printf("Checking %s:\n", file)
		secondaryKeys, err := loadKeysFromLocale(file)
		if err != nil {
			fmt.Printf("  - ❌ Error loading %s: %v\n", file, err)
			hasMissingKeys = true
			continue
		}

		missingFound := false
		var missingKeys []string
		for key := range primaryKeys {
			if _, exists := secondaryKeys[key]; !exists {
				missingKeys = append(missingKeys, key)
			}
		}

		sort.Strings(missingKeys)
		for _, key := range missingKeys {
			fmt.Printf("  - Missing: %s\n", key)
			missingFound = true
			hasMissingKeys = true
		}

		if !missingFound {
			fmt.Println("  ✨ All keys present.")
		}
	}

	// 6. Report potentially untranslated strings
	fmt.Println("\n--- Checking for Potentially Untranslated Strings ---")
	if len(untranslatedStrings) > 0 {
		// Sort the literals for consistent output
		var sortedLiterals []string
		for literal := range untranslatedStrings {
			sortedLiterals = append(sortedLiterals, literal)
		}
		sort.Strings(sortedLiterals)

		for _, literal := range sortedLiterals {
			paths := untranslatedStrings[literal]
			fmt.Printf("  - Potential: \"%s\" (found in %s:%d)\n", literal, paths[0].Filepath, paths[0].Line)
		}
		// We can treat this as a warning for now and not fail the build.
		// To make it a failure, set hasErrors = true here.
	} else {
		fmt.Println("  ✨ None found.")
	}

	fmt.Println("\n--- Linter Finished ---")
	if hasMissingKeys {
		fmt.Println("❌ Found issues that need to be addressed.")
		os.Exit(1)
	} else if hasOrphanedKeys {
		fmt.Println("⚠️  Found orphaned keys. Please consider removing them.")
	} else {
		fmt.Println("✅ All translation files are consistent!")
	}
}

// findUsedKeys scans all .go files for i18n.T("key") calls.
func findUsedKeys(root string) (map[string]struct{}, error) {
	keys := make(map[string]struct{})
	// Regex to find:
	// 1. i18n.T("some.key")
	// 2. string literals that look like translation keys (e.g., in a slice)
	re := regexp.MustCompile(`i18n\.T\("([^"]+)"|\"([a-z]+\.[a-z\._]+)\"`)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		// Exclude the tools directory from the scan
		if info.IsDir() && info.Name() == "tools" {
			return filepath.SkipDir
		}

		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			matches := re.FindAllStringSubmatch(string(content), -1)
			for _, match := range matches {
				// match[1] is from i18n.T(), match[2] is from the general string literal
				if len(match) > 1 && match[1] != "" {
					keys[match[1]] = struct{}{}
				} else if len(match) > 2 && match[2] != "" {
					keys[match[2]] = struct{}{}
				}
			}
		}
		return nil
	})

	return keys, err
}

// findUntranslatedStrings scans for hardcoded strings that might need translation.
func findUntranslatedStrings(root string, usedKeys, allKeys map[string]struct{}) (map[string][]Location, error) {
	untranslated := make(map[string][]Location)
	// Regex to find string literals inside functions that are likely to produce user-facing output.
	re := regexp.MustCompile(`([a-zA-Z0-9_]+\.)?([a-zA-Z0-9_]+)\("([^"]+)"`)
	// Blacklist of function names to ignore.
	blacklist := map[string]struct{}{"Print": {}, "Println": {}, "Printf": {}, "Fatal": {}, "Fatalf": {}, "WriteString": {}}
	keyRe := regexp.MustCompile(`^[a-z_]+\.[a-z\._]+$`)

	// Precompile regexes used in the loop to avoid repeated compilation.
	reAllCaps := regexp.MustCompile(`^[A-Z_]+$`)
	reFormatString := regexp.MustCompile(`^[\s%.,:;()#\d\w-]*%[\s\w-]*$`)

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Exclude the tools directory from the scan
		if info.IsDir() && info.Name() == "tools" {
			return filepath.SkipDir
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") && !strings.HasSuffix(path, "_test.go") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			// Split content into lines to check for log calls
			lines := strings.Split(string(content), "\n")
			for i, line := range lines {
				matches := re.FindAllStringSubmatch(line, -1)
				for _, match := range matches {
					if len(match) < 4 {
						continue
					}
					funcName := match[2]
					literal := match[3]

					if _, isBlacklisted := blacklist[funcName]; isBlacklisted {
						continue
					}

					// Heuristics to filter out false positives:
					// 1. Ignore if it's a known translation key.
					if _, exists := allKeys[literal]; exists {
						continue
					}
					// 2. Ignore if it looks like a translation key.
					if keyRe.MatchString(literal) {
						continue
					}
					// 3. Ignore short or non-text-like strings.
					if len(literal) < 4 {
						continue
					}
					// 4. Ignore if it's just a format specifier or other code artifact.
					if strings.HasPrefix(literal, "file:") || strings.HasPrefix(literal, "http") {
						continue
					}

					// 5. Ignore if it looks like an SQL query.
					upperLiteral := strings.ToUpper(literal)
					sqlKeywords := []string{"SELECT ", "INSERT ", "UPDATE ", "DELETE ", "TRUNCATE ", "PRAGMA ", "CREATE ", "ALTER ", "DROP "}
					isSQL := false
					for _, keyword := range sqlKeywords {
						if strings.HasPrefix(upperLiteral, keyword) {
							isSQL = true
							break
						}
					}
					if isSQL {
						continue
					}

					// 6. Ignore if it's a Go time layout string.
					if strings.HasPrefix(literal, "2006-") {
						continue
					}

					// 7. Ignore if it's an all-caps action constant (e.g., ADD_ACCOUNT).
					if reAllCaps.MatchString(literal) {
						continue
					}

					// 8. Ignore if it's likely just a format string with no real text.
					if reFormatString.MatchString(literal) && !strings.Contains(literal, " ") {
						continue
					}

					untranslated[literal] = append(untranslated[literal], Location{Filepath: path, Line: i + 1})
				}
			}
		}
		return nil
	})

	return untranslated, err
}

// loadKeysFromLocale reads a YAML file and returns a flat map of its keys.
func loadKeysFromLocale(path string) (map[string]struct{}, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := yaml.Unmarshal(content, &data); err != nil {
		return nil, err
	}

	keys := make(map[string]struct{})
	flattenYAML("", data, keys)
	return keys, nil
}

// flattenYAML is a recursive function to convert a nested map into a flat
// map with dot-separated keys.
func flattenYAML(prefix string, node interface{}, keys map[string]struct{}) {
	switch v := node.(type) {
	case map[string]interface{}:
		for k, val := range v {
			newPrefix := k
			if prefix != "" {
				newPrefix = prefix + "." + k
			}
			flattenYAML(newPrefix, val, keys)
		}
	case []interface{}:
		// We don't expect arrays of keys in our structure, but handle it just in case.
		for i, val := range v {
			newPrefix := fmt.Sprintf("%s[%d]", prefix, i)
			flattenYAML(newPrefix, val, keys)
		}
	default:
		// This is a leaf node, add the key.
		if prefix != "" {
			keys[prefix] = struct{}{}
		}
	}
}
