package handler

import (
	"fmt"
	"strings"

	v1 "k8s.io/api/core/v1"
	batch "volcano.sh/apis/pkg/apis/batch/v1alpha1"

	"github.com/raids-lab/crater/dao/model"
)

const (
	diagnosisConfidenceHigh = "high"
	diagnosisSeverityError  = "error"
	diagnosisSeverityWarn   = "warning"
	diagnosisSeverityCrit   = "critical"

	exitCodeSegmentationFault = 139
	exitCodeCommandNotFound   = 127
	exitCodeGracefulTerm      = 143
	maxEvidenceEventsLimit    = 5
)

type ClassifyResult struct {
	TypeName string
	Sample   string
}

type DiagnosisResp struct {
	JobName    string `json:"jobName"`
	Status     string `json:"status"`
	Category   string `json:"category"`
	Diagnosis  string `json:"diagnosis"`
	Solution   string `json:"solution"`
	Confidence string `json:"confidence"`
	Severity   string `json:"severity"`
	Evidence   struct {
		ExitCode   int32    `json:"exitCode,omitempty"`
		ExitReason string   `json:"exitReason,omitempty"`
		Events     []string `json:"events,omitempty"`
	} `json:"evidence"`
}

// CategorizeFailure classifies a failed job without depending on the removed diagnostics routes.
//
//nolint:gocyclo // Rule matching intentionally keeps failure classification in one place.
func CategorizeFailure(job *model.Job) ClassifyResult {
	if job.TerminatedStates != nil {
		terminated := job.TerminatedStates.Data()
		for i := range terminated {
			ts := &terminated[i]
			if strings.EqualFold(ts.Reason, "OOMKilled") {
				return ClassifyResult{TypeName: "OOMKilled", Sample: sampleTerminated(ts)}
			}
			if ts.ExitCode == exitCodeSegmentationFault {
				return ClassifyResult{TypeName: "SegmentationFault", Sample: sampleTerminated(ts)}
			}
			if ts.ExitCode == exitCodeCommandNotFound {
				return ClassifyResult{TypeName: "CommandNotFound", Sample: sampleTerminated(ts)}
			}
			if ts.ExitCode == exitCodeGracefulTerm {
				return ClassifyResult{TypeName: "GracefulTermination", Sample: sampleTerminated(ts)}
			}
		}
	}
	if job.Events != nil {
		events := job.Events.Data()
		for i := range events {
			ev := &events[i]
			if ev.Reason == "ErrImagePull" || ev.Reason == "ImagePullBackOff" {
				return ClassifyResult{TypeName: "ImagePullError", Sample: sampleEvent(ev)}
			}
			if ev.Reason == "FailedScheduling" {
				msg := strings.ToLower(ev.Message)
				switch {
				case strings.Contains(msg, "insufficient"):
					return ClassifyResult{TypeName: "SchedulingInsufficientResources", Sample: sampleEvent(ev)}
				case strings.Contains(msg, "didn't match node selector") || strings.Contains(msg, "node(s) didn't match"):
					return ClassifyResult{TypeName: "SchedulingNodeSelectorMismatch", Sample: sampleEvent(ev)}
				case strings.Contains(msg, "taint"):
					return ClassifyResult{TypeName: "SchedulingTaintMismatch", Sample: sampleEvent(ev)}
				default:
					return ClassifyResult{TypeName: "SchedulingFailed", Sample: sampleEvent(ev)}
				}
			}
			if ev.Reason == "CrashLoopBackOff" ||
				(ev.Reason == "BackOff" && strings.Contains(strings.ToLower(ev.Message), "back-off restarting failed container")) {
				return ClassifyResult{TypeName: "CrashLoopBackOff", Sample: sampleEvent(ev)}
			}
			if ev.Reason == "Evicted" {
				return ClassifyResult{TypeName: "Evicted", Sample: sampleEvent(ev)}
			}
			if ev.Reason == "FailedMount" || strings.Contains(strings.ToLower(ev.Message), "mountvolume") {
				return ClassifyResult{TypeName: "VolumeMountFailed", Sample: sampleEvent(ev)}
			}
			if ev.Reason == "DeadlineExceeded" {
				return ClassifyResult{TypeName: "JobDeadlineExceeded", Sample: sampleEvent(ev)}
			}
		}
	}
	if job.TerminatedStates != nil {
		terminated := job.TerminatedStates.Data()
		for i := range terminated {
			ts := &terminated[i]
			if strings.EqualFold(ts.Reason, "Error") && ts.ExitCode != 0 {
				return ClassifyResult{TypeName: "ContainerError", Sample: sampleTerminated(ts)}
			}
		}
	}
	switch job.Status {
	case batch.Aborted, batch.Terminated:
		return ClassifyResult{TypeName: "JobAbortedOrTerminated"}
	}
	return ClassifyResult{TypeName: "UnknownFailure"}
}

//nolint:gocyclo // Rule-driven diagnosis intentionally keeps category handling in one switch.
func PerformDiagnosis(job *model.Job) DiagnosisResp {
	resp := DiagnosisResp{JobName: job.JobName, Status: string(job.Status)}
	result := CategorizeFailure(job)
	resp.Category = result.TypeName

	switch result.TypeName {
	case "OOMKilled":
		resp.Diagnosis = "作业因内存溢出（OOM）被终止"
		resp.Solution = "建议增加内存请求和限制，优化代码内存占用，并检查是否存在内存泄漏。"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityCrit
	case "ImagePullError":
		resp.Diagnosis = "镜像拉取失败"
		resp.Solution = "建议检查镜像名称、标签、仓库认证、网络连接，并确认镜像是否存在。"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityError
	case "SchedulingInsufficientResources":
		resp.Diagnosis = "集群资源不足，无法调度"
		resp.Solution = "建议降低资源请求，等待其他作业释放资源，或联系管理员扩容。"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityWarn
	case "SchedulingNodeSelectorMismatch":
		resp.Diagnosis = "节点选择器不匹配"
		resp.Solution = "建议检查节点标签配置，修改作业节点选择器，或联系管理员确认可用节点。"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityError
	case "SchedulingTaintMismatch":
		resp.Diagnosis = "节点污点与容忍度不匹配"
		resp.Solution = "建议检查节点 taint，在作业配置中补充 tolerations，或确认调度策略。"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityError
	case "SchedulingFailed":
		resp.Diagnosis = "作业调度失败"
		resp.Solution = "建议查看 FailedScheduling 事件原文，并检查资源请求、节点选择器与污点容忍。"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityError
	case "CrashLoopBackOff":
		resp.Diagnosis = "容器持续崩溃重启"
		resp.Solution = "建议查看容器日志，检查启动命令、配置文件和资源限制。"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityCrit
	case "VolumeMountFailed":
		resp.Diagnosis = "存储卷挂载失败"
		resp.Solution = "建议检查 PVC/PV 绑定状态、存储类、访问模式、挂载路径和权限。"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityError
	case "JobDeadlineExceeded":
		resp.Diagnosis = "作业超出截止时间被终止"
		resp.Solution = "建议评估并调大 activeDeadlineSeconds，或优化作业耗时。"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityWarn
	case "CommandNotFound":
		resp.Diagnosis = "启动命令或文件不存在"
		resp.Solution = "建议检查启动命令、镜像内目标文件/脚本、工作目录和 PATH 配置。"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityError
	case "GracefulTermination":
		resp.Diagnosis = "作业收到终止信号并退出"
		resp.Solution = "建议结合事件时间线确认是否为人工停止、调度回收或平台策略触发。"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityWarn
	case "Evicted":
		resp.Diagnosis = "作业所在 Pod 被节点驱逐"
		resp.Solution = "建议查看节点资源压力与驱逐原因，检查请求/限制是否合理。"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityWarn
	case "SegmentationFault":
		resp.Diagnosis = "段错误（Segmentation Fault）"
		resp.Solution = "建议检查非法内存访问、依赖兼容性、底层算子或 C/C++ 扩展代码。"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityCrit
	case "JobAbortedOrTerminated":
		resp.Diagnosis = "作业被中止或终止"
		resp.Solution = "建议检查是否有人工停止或控制器回收，并结合事件确认触发原因。"
		resp.Confidence = diagnosisConfidenceHigh
		resp.Severity = diagnosisSeverityWarn
	default:
		resp.Diagnosis = "未能自动诊断出具体原因"
		resp.Solution = "建议查看作业日志和 Kubernetes 事件进行人工分析。"
		resp.Confidence = "low"
		resp.Severity = diagnosisSeverityError
	}

	if job.TerminatedStates != nil && len(job.TerminatedStates.Data()) > 0 {
		ts := job.TerminatedStates.Data()[0]
		resp.Evidence.ExitCode = ts.ExitCode
		resp.Evidence.ExitReason = ts.Reason
		if title, suggestion, ok := exitCodeDiagnosis(ts.ExitCode); ok &&
			(resp.Category == "ContainerError" || resp.Category == "UnknownFailure") {
			resp.Diagnosis = fmt.Sprintf("退出异常（Exit %d：%s）", ts.ExitCode, title)
			resp.Solution = suggestion
			resp.Confidence = "medium"
		}
	}
	if job.Events != nil {
		events := job.Events.Data()
		for i := range events {
			if events[i].Type == "Warning" || events[i].Type == "Error" {
				resp.Evidence.Events = append(resp.Evidence.Events, events[i].Message)
			}
		}
		if len(resp.Evidence.Events) > maxEvidenceEventsLimit {
			resp.Evidence.Events = resp.Evidence.Events[:maxEvidenceEventsLimit]
		}
	}
	return resp
}

func sampleTerminated(ts *v1.ContainerStateTerminated) string {
	if ts == nil {
		return ""
	}
	return strings.TrimSpace(ts.Reason + " " + ts.Message)
}

func sampleEvent(ev *v1.Event) string {
	if ev == nil {
		return ""
	}
	return strings.TrimSpace(ev.Message)
}

func exitCodeDiagnosis(exitCode int32) (title, suggestion string, ok bool) {
	mapping := map[int32]struct {
		title      string
		suggestion string
	}{
		1:   {"应用错误", "容器因应用程序错误而停止。建议优先查看日志中的错误堆栈，排查代码异常、依赖缺失、路径错误。"},
		126: {"命令调用错误", "无法调用镜像中指定命令。请确认命令路径正确，且具备可执行权限。"},
		127: {"命令或文件不存在", "找不到镜像中指定命令或文件。请检查启动命令、工作目录、文件路径与镜像内容是否一致。"},
		137: {"立即终止（SIGKILL）", "通常为内存不足或被系统强制终止。建议增加内存申请并结合日志确认是否发生 OOM。"},
		139: {"分段错误（SIGSEGV）", "进程发生非法内存访问。请检查依赖兼容性、底层算子或 C/C++ 扩展代码。"},
		143: {"优雅终止（SIGTERM）", "进程收到终止信号后退出，可能是调度或人工停止触发。请结合事件时间线判断是否为预期行为。"},
	}
	item, exists := mapping[exitCode]
	if !exists {
		return "", "", false
	}
	return item.title, item.suggestion, true
}
