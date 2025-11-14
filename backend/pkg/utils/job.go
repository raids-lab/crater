package utils

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

const (
	maxJobNameLength   = 63
	maxBaseNameLength  = 45
	randomSuffixLength = 5
)

// GenerateJobName generates a DNS-compliant job name with type prefix
// Format: <typePrefix>-<username>-<YYMMDD>-<random>
// Total length is guaranteed to be <= maxBaseNameLength (45 chars)
func GenerateJobName(typePrefix, username string) string {
	dateStr := FormatDateYYMMDD(GetLocalTime())
	randomStr := uuid.New().String()[:randomSuffixLength]

	// Calculate available space for username
	// Format: <typePrefix>-<username>-<YYMMDD>-<random>
	// Fixed parts: 3 hyphens (3) + dateStr (6) + random (5) = 14 chars
	fixedLength := len(typePrefix) + 14
	maxUsernameLength := maxBaseNameLength - fixedLength

	// Truncate username if necessary
	truncatedUsername := username
	if len(username) > maxUsernameLength {
		truncatedUsername = username[:maxUsernameLength]
	}

	// Ensure username ends with alphanumeric (RFC 1035 requirement)
	truncatedUsername = strings.TrimRight(truncatedUsername, "-")
	jobName := fmt.Sprintf("%s-%s-%s-%s", typePrefix, truncatedUsername, dateStr, randomStr)

	return jobName
}
