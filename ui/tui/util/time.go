// Copyright (c) 2026 Keymaster Team
// Keymaster - SSH key management system
// This source code is licensed under the MIT license found in the LICENSE file.
package util

import "time"

const (
	timeLayout1 string = "2006.01.02 15:04:05"
	timeLayout2 string = "2006.01.02 15:04"
	timeLayout3 string = "2006.01.02"
	timeLayout4 string = "02.01.2006 15:04:05"
	timeLayout5 string = "02.01.2006 15:04"
	timeLayout6 string = "02.01.2006"
)

func ParseTime(timeStr string) (time.Time, error) {
	var result time.Time
	var err error
	for _, layout := range []string{timeLayout1, timeLayout2, timeLayout3, timeLayout4, timeLayout5, timeLayout6} {
		result, err = time.Parse(layout, timeStr)
		if err == nil {
			break
		}
	}

	return result, err
}

func StringifyTime(value time.Time) string {
	return value.Format(timeLayout1)
}
