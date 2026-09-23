package utils

import (
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"
	scheduling "volcano.sh/apis/pkg/apis/scheduling/v1beta1"

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

func GetJobRecordExplicitNodeNames(record *model.Job) sets.Set[string] {
	if record == nil {
		return nil
	}
	job := record.Attributes.Data()
	return GetJobExplicitNodeNames(job)
}

func GetJobExplicitNodeNames(job *batch.Job) sets.Set[string] {
	if job == nil || len(job.Spec.Tasks) == 0 {
		return nil
	}

	// Only definite hostname constraints are extracted; ambiguous cases stay wildcard.
	nodes := sets.New[string]()
	for i := range job.Spec.Tasks {
		taskNodes := getPodSpecExplicitNodeNames(&job.Spec.Tasks[i].Template.Spec)
		if taskNodes.Len() == 0 {
			return nil
		}
		nodes = nodes.Union(taskNodes)
	}
	if nodes.Len() == 0 {
		return nil
	}
	return nodes
}

//nolint:gocyclo // basic tool function
func getPodSpecExplicitNodeNames(spec *v1.PodSpec) sets.Set[string] {
	if spec == nil {
		return nil
	}

	nodes := sets.New[string]()
	if nodeName := spec.NodeSelector[v1.LabelHostname]; nodeName != "" {
		nodes.Insert(nodeName)
	}

	if spec.Affinity == nil || spec.Affinity.NodeAffinity == nil ||
		spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution == nil {
		return nodes
	}
	terms := spec.Affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
	if len(terms) == 0 {
		return nodes
	}

	affinityNodes := sets.New[string]()
	hasWildcardTerm := false
	for i := range terms {
		// Node selector terms are ORed, while repeated hostname expressions in a term are ANDed.
		termNodes := sets.New[string]()
		hasHostname := false
		for j := range terms[i].MatchExpressions {
			expr := terms[i].MatchExpressions[j]
			if expr.Key != v1.LabelHostname {
				continue
			}
			if expr.Operator != v1.NodeSelectorOpIn || len(expr.Values) == 0 {
				return nil
			}
			exprNodes := sets.New(expr.Values...).Delete("")
			if exprNodes.Len() == 0 {
				return nil
			}
			if !hasHostname {
				termNodes = exprNodes
				hasHostname = true
				continue
			}
			termNodes = termNodes.Intersection(exprNodes)
		}
		if !hasHostname {
			// Kubernetes ORs selector terms, so a term without hostname makes affinity wildcard here.
			hasWildcardTerm = true
			continue
		}
		affinityNodes = affinityNodes.Union(termNodes)
	}

	if nodes.Len() > 0 {
		if affinityNodes.Len() > 0 && !hasWildcardTerm {
			return nodes.Intersection(affinityNodes)
		}
		return nodes
	}
	if hasWildcardTerm {
		return nil
	}
	return affinityNodes
}

// NodeConstraintsOverlap treats an empty set as "any node", so a wildcard job overlaps everything.
func NodeConstraintsOverlap(left, right sets.Set[string]) bool {
	if left.Len() == 0 || right.Len() == 0 {
		return true
	}
	return left.Intersection(right).Len() > 0
}

// CanResourceDomainBlock determines if a job with candidateDomain can be blocked by a timed out job with timedOutDomain
func CanResourceDomainBlock(timedOutDomain, candidateDomain string) bool {
	if timedOutDomain == ResourceDomainCPUOnly {
		return candidateDomain == ResourceDomainCPUOnly
	}
	return timedOutDomain == candidateDomain || candidateDomain == ResourceDomainCPUOnly
}

// IsPodGroupAdmitted reports whether volcano let the pod group into a queue. The empty phase covers a
// job whose pod group does not exist yet, which is indistinguishable from one still pending.
func IsPodGroupAdmitted(phase scheduling.PodGroupPhase) bool {
	return phase != "" && phase != scheduling.PodGroupPending
}

// IsJobInqueue reports the window volcano keeps a vcjob Pending after admitting its pod group.
func IsJobInqueue(jobPhase batch.JobPhase, podGroupPhase scheduling.PodGroupPhase) bool {
	return (jobPhase == "" || jobPhase == batch.Pending) && IsPodGroupAdmitted(podGroupPhase)
}

// IsPodGroupWaitingForNodes reports whether volcano admitted the pod group but has not yet bound
// its minimum members, so the job still competes for nodes and can starve.
func IsPodGroupWaitingForNodes(phase scheduling.PodGroupPhase) bool {
	return phase == scheduling.PodGroupInqueue || phase == scheduling.PodGroupUnknown
}

// IsJobPhaseTerminal reports whether the vcjob reached a phase it never leaves.
func IsJobPhaseTerminal(phase batch.JobPhase) bool {
	switch phase {
	case batch.Completed, batch.Failed, batch.Aborted, batch.Terminated:
		return true
	default:
		return false
	}
}
