//nolint:gocritic,gocyclo // Policy evaluation intentionally operates on full snapshot values in a single rules engine.
package storagegovernance

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/raids-lab/crater/pkg/llm"
)

const (
	decisionEvidenceCapacity         = 5
	usageRatioNearLimit              = 0.95
	usageRatioAlertLimit             = 0.90
	highGPUHistoryPercent    float64 = 80
	lowGPUHistoryPercent     float64 = 10
)

func ApplySafetyConstraints(
	snapshot DecisionSnapshot,
	decision llm.LLMDecisionResponse,
	cfg ConstraintConfig,
	now time.Time,
) (llm.LLMDecisionResponse, ConstraintEvaluation) {
	originalDecision := decision
	finalDecision := decision
	evaluation := ConstraintEvaluation{
		PolicyVersion: cfg.PolicyVersion,
		Violations:    []string{},
		Adjustments:   []string{},
	}

	if snapshot.UsageRatio < cfg.AlertThreshold {
		if finalDecision.AllowExpand || finalDecision.FreezeNewJobs {
			evaluation.Violations = append(evaluation.Violations, "存储使用率低于告警阈值")
			finalDecision.AllowExpand = false
			finalDecision.ExpandBytes = 0
			finalDecision.FreezeNewJobs = false
			evaluation.Adjustments = append(evaluation.Adjustments, "已禁用扩容，因为当前存储使用率低于告警阈值")
		}
	}

	if snapshot.TheoreticalQuotaBytes <= 0 && finalDecision.AllowExpand {
		evaluation.Violations = append(evaluation.Violations, "理论配额为无限制或未设置")
		finalDecision.AllowExpand = false
		finalDecision.ExpandBytes = 0
		evaluation.Adjustments = append(evaluation.Adjustments, "已禁用扩容，因为用户不存在有限的理论配额")
	}

	if snapshot.IsCurrentlyExpanded && finalDecision.AllowExpand {
		evaluation.Violations = append(evaluation.Violations, "用户当前已处于临时扩容状态")
		finalDecision.AllowExpand = false
		finalDecision.ExpandBytes = 0
		evaluation.Adjustments = append(evaluation.Adjustments, "已禁用扩容，因为上一轮临时扩容仍在生效")
	}

	if snapshot.LastExpandAt != nil && now.Sub(*snapshot.LastExpandAt) < cfg.ExpansionCooldown && finalDecision.AllowExpand {
		evaluation.Violations = append(evaluation.Violations, "最近一次扩容仍处于冷却时间窗口内")
		finalDecision.AllowExpand = false
		finalDecision.ExpandBytes = 0
		evaluation.Adjustments = append(evaluation.Adjustments, "已禁用扩容，因为距上次扩容执行尚未超过冷却时间")
	}

	if snapshot.UsageRatio >= 1.0 && finalDecision.AllowExpand {
		evaluation.Violations = append(evaluation.Violations, "用户当前已超过理论配额")
		finalDecision.AllowExpand = false
		finalDecision.ExpandBytes = 0
		evaluation.Adjustments = append(evaluation.Adjustments, "已禁用扩容，因为当前使用量已经超过理论配额")
	}

	if finalDecision.AllowExpand && finalDecision.ExpandBytes <= 0 {
		evaluation.Violations = append(evaluation.Violations, "启用扩容时，扩容量必须为正数")
		finalDecision.AllowExpand = false
		finalDecision.ExpandBytes = 0
		evaluation.Adjustments = append(evaluation.Adjustments, "已禁用扩容，因为建议扩容量不是正数")
	}

	if finalDecision.AllowExpand {
		maxAllowed := int64(math.MaxInt64)

		if snapshot.TheoreticalQuotaBytes > 0 && cfg.MaxExpandRatio > 0 {
			maxByRatio := int64(float64(snapshot.TheoreticalQuotaBytes) * cfg.MaxExpandRatio)
			if maxByRatio > 0 && maxByRatio < maxAllowed {
				maxAllowed = maxByRatio
			}
		}

		if cfg.MaxExpandBytes > 0 && cfg.MaxExpandBytes < maxAllowed {
			maxAllowed = cfg.MaxExpandBytes
		}

		if snapshot.PlatformTotalBytes > 0 {
			reservedByRatio := int64(float64(snapshot.PlatformTotalBytes) * cfg.MinPlatformReservedRatio)
			reservedBytes := maxInt64(cfg.MinPlatformReservedBytes, reservedByRatio)
			maxByPlatform := snapshot.PlatformAvailableBytes - reservedBytes
			if maxByPlatform < maxAllowed {
				maxAllowed = maxByPlatform
			}
		}

		if maxAllowed <= 0 {
			evaluation.Violations = append(evaluation.Violations, "执行扩容将突破平台预留容量下限")
			finalDecision.AllowExpand = false
			finalDecision.ExpandBytes = 0
			evaluation.Adjustments = append(evaluation.Adjustments, "已禁用扩容，因为平台预留容量会低于安全阈值")
		} else if finalDecision.ExpandBytes > maxAllowed {
			evaluation.Violations = append(evaluation.Violations, "建议扩容量超过安全上限")
			evaluation.Adjustments = append(
				evaluation.Adjustments,
				fmt.Sprintf("已将扩容量从 %d 字节收敛到 %d 字节", finalDecision.ExpandBytes, maxAllowed),
			)
			finalDecision.ExpandBytes = maxAllowed
		}
	}

	if cfg.ForceFreezeWhenOverQuota && snapshot.UsageRatio >= 1.0 && !finalDecision.AllowExpand && !finalDecision.FreezeNewJobs {
		evaluation.Violations = append(evaluation.Violations, "用户当前已超过理论配额")
		finalDecision.FreezeNewJobs = true
		evaluation.Adjustments = append(evaluation.Adjustments, "已强制冻结新作业，因为当前使用量已经超过理论配额")
	}

	if !finalDecision.AllowExpand {
		finalDecision.ExpandBytes = 0
	}

	evaluation.Violations = uniqueStrings(evaluation.Violations)
	evaluation.Adjustments = uniqueStrings(evaluation.Adjustments)
	evaluation.Adjusted = len(evaluation.Adjustments) > 0
	evaluation.Blocked = originalDecision.AllowExpand && !finalDecision.AllowExpand
	finalDecision.Reason = rewriteDecisionReason(snapshot, originalDecision, finalDecision, evaluation)

	return finalDecision, evaluation
}

func rewriteDecisionReason(
	snapshot DecisionSnapshot,
	originalDecision llm.LLMDecisionResponse,
	finalDecision llm.LLMDecisionResponse,
	evaluation ConstraintEvaluation,
) string {
	evidenceParts := make([]string, 0, decisionEvidenceCapacity)

	switch {
	case snapshot.UsageRatio >= 1.0:
		evidenceParts = append(evidenceParts, "当前使用量已经超过理论配额")
	case snapshot.UsageRatio >= usageRatioNearLimit:
		evidenceParts = append(evidenceParts, "存储使用率已逼近阈值")
	case snapshot.UsageRatio >= usageRatioAlertLimit:
		evidenceParts = append(evidenceParts, "存储使用率接近阈值")
	default:
		evidenceParts = append(evidenceParts, "当前使用量已回落到安全区间")
	}

	if growthPhrase := describeGrowth(snapshot.GrowthRateBytesPerHour); growthPhrase != "" {
		evidenceParts = append(evidenceParts, growthPhrase)
	}

	if snapshot.GPUDataAvailable {
		switch {
		case snapshot.MaxGPUHistoryPercent >= highGPUHistoryPercent:
			evidenceParts = append(evidenceParts, "GPU 历史峰值较高")
		case snapshot.MaxGPUHistoryPercent <= lowGPUHistoryPercent:
			evidenceParts = append(evidenceParts, "GPU 历史峰值不足以支撑高价值训练判断")
		}
	}

	if snapshot.IsCurrentlyExpanded {
		evidenceParts = append(evidenceParts, "用户当前已处于临时扩容状态")
	}

	switch snapshot.ShrinkStage {
	case "expanded":
		evidenceParts = append(evidenceParts, "当前处于临时扩容后的观察阶段")
	case "buffer_reduction":
		evidenceParts = append(evidenceParts, "当前处于缩容缓冲阶段")
	}

	baseReason := strings.Join(uniqueStrings(evidenceParts), "，")
	actionClause := describeFinalAction(snapshot, originalDecision, finalDecision)
	if baseReason == "" {
		baseReason = actionClause
	} else {
		baseReason = baseReason + "，" + actionClause
	}

	if evaluation.Adjusted {
		return appendConstraintReason(baseReason, evaluation)
	}
	return baseReason
}

func describeGrowth(growthRate *float64) string {
	if growthRate == nil {
		return ""
	}

	const gib = 1024 * 1024 * 1024
	switch {
	case *growthRate >= 4*gib:
		return "增长速度较快"
	case *growthRate > 0:
		return "增长速率平稳"
	case *growthRate <= -0.5*gib:
		return "使用量持续下降"
	default:
		return "使用量整体稳定"
	}
}

func describeFinalAction(
	snapshot DecisionSnapshot,
	originalDecision llm.LLMDecisionResponse,
	finalDecision llm.LLMDecisionResponse,
) string {
	if finalDecision.AllowExpand {
		if originalDecision.AllowExpand && originalDecision.ExpandBytes != finalDecision.ExpandBytes {
			return fmt.Sprintf("最终建议在安全约束下扩容 %d 字节，以保护当前作业写入", finalDecision.ExpandBytes)
		}
		return fmt.Sprintf("建议扩容 %d 字节，以保护当前作业写入", finalDecision.ExpandBytes)
	}

	if finalDecision.FreezeNewJobs {
		return "不应继续扩容，并需要冻结新作业"
	}

	switch {
	case snapshot.ShrinkStage == "buffer_reduction":
		return "无需继续扩容，可以进一步恢复到原始配额"
	case snapshot.IsCurrentlyExpanded && snapshot.UsageRatio < 0.9:
		return "无需继续扩容，更适合进入分阶段缩容观察"
	default:
		return "不应继续扩容，建议继续观察"
	}
}

func appendConstraintReason(reason string, evaluation ConstraintEvaluation) string {
	suffix := "安全约束："
	if len(evaluation.Adjustments) > 0 {
		suffix += " " + joinWithSemicolon(evaluation.Adjustments)
	}
	if len(evaluation.Violations) > 0 {
		suffix += " | 违规项：" + joinWithSemicolon(evaluation.Violations)
	}
	if reason == "" {
		return suffix
	}
	return reason + " | " + suffix
}

func joinWithSemicolon(items []string) string {
	return strings.Join(uniqueStrings(items), "; ")
}

func uniqueStrings(items []string) []string {
	if len(items) == 0 {
		return items
	}

	seen := make(map[string]struct{}, len(items))
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		out = append(out, trimmed)
	}
	return out
}

func maxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
