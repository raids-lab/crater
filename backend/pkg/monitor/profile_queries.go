package monitor

import (
	"fmt"
	"regexp"
	"strings"
)

// Prometheus anchors label regex matchers; quote pod names before joining them.
func profilePodPattern(podNames []string) string {
	patterns := make([]string, len(podNames))
	for i, name := range podNames {
		patterns[i] = regexp.QuoteMeta(name)
	}
	return strings.Join(patterns, "|")
}

func profileResourceQuery(metric, resource, namespace, pods, window string) string {
	return fmt.Sprintf("sum(max by (pod, container) (max_over_time(%s{namespace=%q,pod=~%q,resource=%q}[%s])))",
		metric, namespace, pods, resource, window)
}
