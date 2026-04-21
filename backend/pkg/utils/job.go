package utils

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"

	"github.com/raids-lab/crater/dao/model"
)

const (
	maxJobNameLength      = 63
	maxBaseNameLength     = 45
	randomSuffixLength    = 5
	ResourceDomainCPUOnly = "cpu-only"
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

func IsSingleNodeJob(record *model.Job) bool {
	if record == nil || record.Attributes.Data() == nil || len(record.Attributes.Data().Spec.Tasks) != 1 {
		return false
	}
	return record.Attributes.Data().Spec.Tasks[0].Replicas == 1
}

func GetResourceDomain(resources v1.ResourceList) string {
	domains := make([]string, 0)
	for name, quantity := range resources {
		if quantity.IsZero() || !strings.Contains(name.String(), "/") {
			continue
		}
		domains = append(domains, name.String())
	}
	if len(domains) == 0 {
		return ResourceDomainCPUOnly
	}
	sort.Strings(domains)
	return strings.Join(domains, ",")
}

func GetJobResourceDomain(record *model.Job) string {
	if record == nil {
		return ResourceDomainCPUOnly
	}
	return GetResourceDomain(record.Resources.Data())
}

// CanResourceDomainBlock determines if a job with candidateDomain can be blocked by a timed out job with timedOutDomain
func CanResourceDomainBlock(timedOutDomain, candidateDomain string) bool {
	if timedOutDomain == ResourceDomainCPUOnly {
		return candidateDomain == ResourceDomainCPUOnly
	}
	return timedOutDomain == candidateDomain || candidateDomain == ResourceDomainCPUOnly
}
