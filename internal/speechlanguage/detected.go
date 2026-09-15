package speechlanguage

import (
	"slices"
	"strings"
)

// MergeDetected retains evidence across finalized turns. Overflow is represented
// as multiple languages (ISO 639 mul), never silently classified as English.
func MergeDetected(left, right []string) []string {
	result := slices.Clone(left)
	for _, value := range right {
		value = strings.TrimSpace(value)
		if Unspecified(value) || Validate(value) != nil || slices.Contains(result, value) {
			continue
		}
		if len(result) >= 8 {
			result[7] = "mul"
			continue
		}
		result = append(result, value)
	}
	return result
}
