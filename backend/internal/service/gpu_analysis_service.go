// 请将此文件保存或替换为 internal/service/gpu_analysis_service.go
package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"errors" // 用于 gorm.ErrRecordNotFound

	"gorm.io/gorm" // 用于 gorm.ErrRecordNotFound
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/pkg/monitor"
	"github.com/raids-lab/crater/pkg/prompts"
	"github.com/raids-lab/crater/pkg/prompts/gpu_analysis"
)

const (
	PodAnalysisMinAge    = 2 * time.Hour
	MetricsQueryDuration = 2 * time.Hour
	LLMRequestTimeout    = 300 * time.Second  // 为LLM请求设置超时
	MaxQueryLength       = 4000               // LLM查询的最大长度
	SleepBetweenRetries  = 2 * time.Second    // LLM请求失败时的重试间隔
	GpuAnalysisMaxAge    = 7 * 24 * time.Hour // GPU分析记录的最大保留时间
	psColumnNum          = 5                  // ps 命令输出的列数
	maxScriptLength      = 20000
)

// LLMResponse 定义了我们期望 LLM 返回的 JSON 结构
// 【新增】: 为第一阶段专门设计的 Response 结构
type LLMResponsePhase1 struct {
	PID    int    `json:"pid"` // LLM 识别出的可疑进程 PID
	Score  int    `json:"score"`
	Reason string `json:"reason"`
}

// 【修改】: 第二阶段的 Response 结构（原 LLMResponse）
type LLMResponsePhase2 struct {
	Score  int    `json:"score"`
	Reason string `json:"reason"`
}

type GpuAnalysisService struct {
	q              *query.Query
	kubeConfig     *rest.Config
	kubeClient     kubernetes.Interface
	promClient     *monitor.PrometheusClient
	cfgService     *ConfigService
	httpClient     *http.Client        // 添加一个可复用的HTTP客户端
	jobQueue       chan *model.Job     // 用于异步分析的 Job 任务队列
	processingJobs map[string]struct{} // 用于跟踪已在队列中或正在处理的 Job
	processingLock sync.Mutex          // 用于保护 processingJobs 的并发访问
}

func NewGpuAnalysisService(
	q *query.Query,
	kubeConfig *rest.Config,
	kubeClient kubernetes.Interface,
	promClient *monitor.PrometheusClient,
	cfgService *ConfigService,
) *GpuAnalysisService {
	jobQueue := make(chan *model.Job, MaxQueryLength)
	s := &GpuAnalysisService{
		q:              q,
		kubeConfig:     kubeConfig,
		kubeClient:     kubeClient,
		promClient:     promClient,
		cfgService:     cfgService,
		httpClient:     &http.Client{Timeout: LLMRequestTimeout}, // 初始化HTTP客户端并设置超时
		jobQueue:       jobQueue,
		processingJobs: make(map[string]struct{}),
	}
	go s.startAnalysisWorker() // 启动后台工作线程
	return s
}

func (s *GpuAnalysisService) AnalyzePodByName(ctx context.Context, namespace, podName string) (*model.GpuAnalysis, error) {
	// 1. 从 Kubernetes 获取最新的 Pod 对象
	pod, err := s.kubeClient.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get pod %s in namespace %s: %w", podName, namespace, err)
	}

	// 2. 调用核心分析逻辑
	// 注意：为 demo 方便，手动触发时我们不检查 pod 运行时长和锁定状态
	return s.performFullAnalysis(ctx, pod)
}

// AnalyzeJobByName 是分析 Job 的新入口点
func (s *GpuAnalysisService) AnalyzeJobByName(ctx context.Context, jobName string) (*model.GpuAnalysis, error) {
	// 1. 查找 Job 并验证其是否符合分析条件
	j := s.q.Job
	job, err := j.WithContext(ctx).Preload(j.User).Where(j.JobName.Eq(jobName)).First()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("job '%s' not found in database", jobName)
		}
		return nil, fmt.Errorf("failed to query job '%s': %w", jobName, err)
	}

	// 检查 Job 状态是否为运行中
	// 注意: 请根据您的 Job 模型中的实际运行状态调整此处的常量
	if !IsEligibleForGpuAnalysis(job) {
		return nil, fmt.Errorf("job '%s' is not eligible for GPU analysis (status: %s)", jobName, job.Status)
	}

	// 2. 查找与该 Job 关联的正在运行的 Pod
	// 使用 job-name 标签来定位 Pod
	podList, err := s.kubeClient.CoreV1().Pods(job.Attributes.Data().Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("volcano.sh/job-name=%s", job.JobName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list pods for job '%s': %w", jobName, err)
	}

	var targetPod *v1.Pod
	for i := range podList.Items {
		pod := &podList.Items[i]
		if pod.Status.Phase == v1.PodRunning {
			targetPod = pod
			break // 找到第一个 running 的 pod 即可
		}
	}

	if targetPod == nil {
		return nil, fmt.Errorf("no running pod found for job '%s'", jobName)
	}

	klog.Infof("Found running pod '%s' for job '%s', proceeding with analysis.", targetPod.Name, jobName)

	// 3. 调用核心分析逻辑
	return s.performFullAnalysis(ctx, targetPod)
}

func (s *GpuAnalysisService) TriggerAllJobsAnalysis(ctx context.Context) (int, error) {
	j := s.q.Job

	go s.cleanOvertimeGpuAnalyses(context.Background())

	// 1. 查找所有状态为 "Running" 且未被锁定的 Job
	var eligibleJobs []*model.Job
	// 注意: 这里的 job.Status 可能需要根据你的模型调整
	eligibleJobs, err := j.WithContext(ctx).
		Where(j.Status.Eq("Running")).
		Where(j.LockedTimestamp.Eq(time.Time{})).
		Find()

	if err != nil {
		return 0, fmt.Errorf("failed to list eligible jobs for analysis: %w", err)
	}
	klog.Infof("Found %d eligible running jobs for analysis.\n", len(eligibleJobs))

	if len(eligibleJobs) == 0 {
		klog.Info("Triggered all jobs analysis, but found no eligible running jobs.")
		return 0, nil
	}
	var gpuJobs []*model.Job
	for _, job := range eligibleJobs {
		if IsEligibleForGpuAnalysis(job) {
			gpuJobs = append(gpuJobs, job)
		}
	}

	if len(gpuJobs) == 0 {
		klog.Info("Found running jobs, but none are using GPU resources.")
		go s.cleanupStaleAnalyses(context.Background())
		return 0, nil
	}

	// 2. 将 Job 名称推入队列
	count := 0
	for _, job := range gpuJobs {
		s.processingLock.Lock() // 加锁
		if _, isProcessing := s.processingJobs[job.JobName]; isProcessing {
			// 如果任务已在处理中，则跳过
			s.processingLock.Unlock() // 解锁
			klog.Infof("Job '%s' is already in the processing queue, skipping.", job.JobName)
			continue
		}
		s.processingJobs[job.JobName] = struct{}{}
		s.processingLock.Unlock() // 解锁
		select {
		case s.jobQueue <- job:
			count++
		default:
			// 如果队列已满，则记录一个警告并停止推送
			klog.Warningf("GPU analysis job queue is full. Stopped enqueuing after %d jobs.", count)
			// 返回已经成功入队的数量
			return count, fmt.Errorf("job queue is full, only %d jobs were enqueued", count)
		}
	}

	klog.Infof("Successfully enqueued %d jobs for asynchronous GPU analysis.", count)
	return count, nil
}

// startAnalysisWorker 是一个后台 goroutine，用于处理异步分析任务
func (s *GpuAnalysisService) startAnalysisWorker() {
	klog.Info("Starting GPU analysis worker...")
	// 使用 for-range 循环会一直阻塞，直到 channel 中有新数据或 channel 被关闭
	for job := range s.jobQueue {
		klog.Infof("Worker picked up job '%s' from queue for analysis.", job.JobName)

		// 为了避免单个任务的panic导致整个worker退出，我们使用 recover
		func() {
			defer func() {
				s.processingLock.Lock()
				delete(s.processingJobs, job.JobName)
				s.processingLock.Unlock()
				klog.Infof("Finished processing job '%s', removed from tracking set.", job.JobName)

				if r := recover(); r != nil {
					klog.Errorf("Recovered from panic during analysis of job %s: %v", job.JobName, r)
				}
			}()

			// 使用一个新的 context，而不是依赖于触发请求的 context
			ctx := context.Background()

			// 调用我们已有的、针对单个Job的分析逻辑
			// 这里我们不关心返回值，因为结果是直接更新到数据库的
			_, err := s.AnalyzeJobByName(ctx, job.JobName)
			if err != nil {
				// 记录错误，但不会阻塞队列中的其他任务
				klog.Errorf("Async analysis for job '%s' failed: %v", job.JobName, err)
			} else {
				klog.Infof("Async analysis for job '%s' completed successfully.", job.JobName)
			}
		}()

		// 可以选择在任务间稍作停顿，避免对 K8s API 和 LLM 造成太大压力
		s.cleanupStaleAnalyses(context.Background())
		time.Sleep(SleepBetweenRetries)
	}
	klog.Info("GPU analysis worker has stopped.") // 正常情况下不应执行到这里
}

// =================================================================
// === 核心分析逻辑 (被手动和自动两种方式复用) ===
func (s *GpuAnalysisService) performFullAnalysis(ctx context.Context, pod *v1.Pod) (*model.GpuAnalysis, error) {
	jobName, ok := pod.Labels["volcano.sh/job-name"]
	if !ok {
		return nil, fmt.Errorf("pod %s is missing 'volcano.sh/job-name' label", pod.Name)
	}
	job, err := s.q.Job.WithContext(ctx).Preload(s.q.Job.User).Where(s.q.Job.JobName.Eq(jobName)).First()
	if err != nil {
		return nil, fmt.Errorf("could not find job record for pod %s: %w", pod.Name, err)
	}

	metrics, err := s.promClient.QueryGpuAnalysisMetrics(pod.Namespace, pod.Name, MetricsQueryDuration)
	if err != nil {
		klog.Warningf("Prometheus query failed for pod %s, but continuing analysis. Error: %v", pod.Name, err)
		metrics = &monitor.GpuAnalysisMetrics{}
	}

	if len(pod.Spec.Containers) == 0 {
		return nil, fmt.Errorf("pod %s has no containers", pod.Name)
	}
	containerName := pod.Spec.Containers[0].Name

	psCommand := `ps -eo pid,ppid,pcpu,pmem,user,cmd --sort=-pmem | head -n 20`
	processList, err := s.execCommandInPod(ctx, pod.Namespace, pod.Name, containerName, []string{"sh", "-c", psCommand})
	if err != nil {
		klog.Warningf("Failed to run ps command in pod %s: %v", pod.Name, err)
	}
	processList = strings.TrimSpace(processList)

	llmConfig, err := s.cfgService.GetLLMConfig(ctx)
	if err != nil || llmConfig.GetChatCompletionURL() == "" {
		return nil, fmt.Errorf("LLM configuration (API Key and URL) is missing or invalid")
	}

	// === 阶段 1: 识别 PID 并初步打分 ===
	// 【新 Prompt】: 将整个进程列表交给 LLM

	prompt1 := s.buildPhase1Prompt(metrics, processList)
	resp1, err := s.callLLMForPhase1(ctx, llmConfig, prompt1)
	if err != nil {
		return nil, fmt.Errorf("LLM Phase 1 call failed for pod %s: %w", pod.Name, err)
	}

	// 从 LLM 的响应中获取 PID 和分数
	resp1.PID = s.makeSurePid(processList, resp1.PID)
	pid := resp1.PID
	phase1Score := resp1.Score
	llmReason1 := resp1.Reason

	// === 阶段 2: 查找脚本并二次打分 ===
	phase2Score := -1
	llmReason2 := ""
	fullCommand, scriptContent, err := s.findAndReadScriptInPod(ctx, pod.Namespace, pod.Name, containerName, pid)
	if phase1Score > 3 && pid > 0 { // 只有在 LLM 找到了可疑 PID 且分数较高时才进行
		if err != nil {
			klog.Warningf("Could not find or read script for pod %s: %v", pod.Name, err)
		}

		if scriptContent != "" {
			prompt2 := s.buildPhase2Prompt(metrics, fullCommand, scriptContent, llmReason1)
			resp2, err := s.callLLMForPhase2(ctx, llmConfig, prompt2)
			if err != nil {
				klog.Errorf("LLM Phase 2 call failed for pod %s: %v", pod.Name, err)
			} else {
				phase2Score = resp2.Score
				llmReason2 = resp2.Reason
			}
		}
	}

	metricsJSON, _ := json.Marshal(metrics)
	analysisRecord := &model.GpuAnalysis{
		JobID:             job.ID,
		JobName:           job.JobName,
		UserID:            job.UserID,
		UserName:          job.User.Name,
		PodName:           pod.Name,
		Namespace:         pod.Namespace,
		Phase1Score:       phase1Score,
		Phase2Score:       phase2Score,
		Phase1LLMReason:   llmReason1,
		Phase2LLMReason:   llmReason2,
		LLMVersion:        llmConfig.ModelName,
		Command:           fullCommand,
		ScriptContent:     scriptContent,
		HistoricalMetrics: string(metricsJSON),
		ReviewStatus:      model.ReviewStatusPending,
	}

	// 保存或更新分析记录
	analysisRecord, err = s.postFullAnalysisHandler(ctx, analysisRecord)
	if err != nil {
		return nil, fmt.Errorf("failed to save analysis record for pod %s: %w", pod.Name, err)
	}

	klog.Infof("Successfully performed and saved analysis for pod: %s (Job: %s). Final Score: %d, %d.",
		pod.Name, jobName, phase1Score, phase2Score)
	return analysisRecord, nil
}

// postFullAnalysisHandler 处理分析结果的保存，并根据业务逻辑清理旧记录
// 逻辑如下：
// 1. 同命令 & 已评审 -> 保留旧记录 (恢复)，丢弃新记录。
// 2. 同命令 & 未评审 -> 硬删除旧记录 (覆盖)，保存新记录。
// 3. 异命令 -> 软删除旧记录 (归档)，保存新记录。
// 1. 定义一个结构体来承载决策结果
type analysisPlan struct {
	idsToHardDelete []uint
	idsToSoftDelete []uint
	recordToRestore *model.GpuAnalysis
	shouldCreateNew bool
	finalRecord     *model.GpuAnalysis
}

// 2. 抽离核心决策逻辑 (降低复杂度的关键)
func (s *GpuAnalysisService) decideAnalysisAction(existingRecords []*model.GpuAnalysis, current *model.GpuAnalysis) analysisPlan {
	plan := analysisPlan{
		shouldCreateNew: true,
		finalRecord:     current,
	}

	if len(existingRecords) == 0 {
		return plan
	}

	latest := existingRecords[0]
	others := existingRecords[1:]

	// 处理旧数据：除了最新的一条，其余全部硬删除
	for _, r := range others {
		plan.idsToHardDelete = append(plan.idsToHardDelete, r.ID)
	}

	isCommandSame := (latest.Command == current.Command)
	isReviewed := (latest.ReviewStatus != model.ReviewStatusPending)

	if isCommandSame {
		if isReviewed {
			// 场景 A: 同命令 & 已评审 -> 保留旧的，不建新的
			plan.recordToRestore = latest
			plan.shouldCreateNew = false
			plan.finalRecord = latest
		} else {
			// 场景 B: 同命令 & 未评审 -> 覆盖旧的
			plan.idsToHardDelete = append(plan.idsToHardDelete, latest.ID)
		}
	} else {
		// 场景 C: 命令不一致 -> 软删除旧的作为历史
		plan.idsToSoftDelete = append(plan.idsToSoftDelete, latest.ID)
	}

	return plan
}

// 3. 主函数变得清爽
func (s *GpuAnalysisService) postFullAnalysisHandler(ctx context.Context, analysisRecord *model.GpuAnalysis) (*model.GpuAnalysis, error) {
	ga := s.q.GpuAnalysis

	// 1. 获取记录
	existingRecords, err := ga.WithContext(ctx).
		Where(ga.JobID.Eq(analysisRecord.JobID)).
		Unscoped().
		Order(ga.ID.Desc()).
		Find()
	if err != nil {
		return nil, fmt.Errorf("error querying existing records: %w", err)
	}

	// 2. 决策
	plan := s.decideAnalysisAction(existingRecords, analysisRecord)

	// 3. 执行事务 (将具体执行逻辑传入)
	err = s.q.Transaction(func(tx *query.Query) error {
		return s.executeAnalysisPlan(ctx, tx, plan, analysisRecord)
	})

	if err != nil {
		return nil, err
	}

	return plan.finalRecord, nil
}

// 4. 抽离事务执行细节
func (s *GpuAnalysisService) executeAnalysisPlan(
	ctx context.Context,
	tx *query.Query,
	plan analysisPlan,
	newRecord *model.GpuAnalysis,
) error {
	ga := tx.GpuAnalysis.WithContext(ctx)

	// 3.1 物理删除
	if len(plan.idsToHardDelete) > 0 {
		if _, err := ga.Unscoped().Where(tx.GpuAnalysis.ID.In(plan.idsToHardDelete...)).Delete(); err != nil {
			return err
		}
	}

	// 3.2 软删除
	if len(plan.idsToSoftDelete) > 0 {
		if _, err := ga.Where(tx.GpuAnalysis.ID.In(plan.idsToSoftDelete...)).Delete(); err != nil {
			return err
		}
	}

	// 3.3 恢复
	if plan.recordToRestore != nil && plan.recordToRestore.DeletedAt.Valid {
		if _, err := ga.Where(tx.GpuAnalysis.ID.Eq(plan.recordToRestore.ID)).
			UpdateSimple(tx.GpuAnalysis.DeletedAt.Null()); err != nil {
			return err
		}
		plan.recordToRestore.DeletedAt = gorm.DeletedAt{}
	}

	// 3.4 创建
	if plan.shouldCreateNew {
		if err := ga.Create(newRecord); err != nil {
			return err
		}
	}

	return nil
}

func (s *GpuAnalysisService) callLLMForPhase1(ctx context.Context, cfg *LLMConfig, prompt string) (*LLMResponsePhase1, error) {
	systemPrompt, err := gpu_analysis.GetSystemPrompt(map[string]any{})
	if err != nil {
		klog.Errorf("Error building Phase 1 system prompt: %v", err)
		return nil, err
	}
	return prompts.CallLLMAPI[LLMResponsePhase1](
		s.httpClient,
		ctx,
		cfg.GetChatCompletionURL(),
		cfg.APIKey,
		cfg.ModelName,
		systemPrompt,
		prompt,
	)
}

func (s *GpuAnalysisService) callLLMForPhase2(ctx context.Context, cfg *LLMConfig, prompt string) (*LLMResponsePhase2, error) {
	systemPrompt, err := gpu_analysis.GetSystemPrompt(map[string]any{})
	if err != nil {
		klog.Errorf("Error building Phase 2 system prompt: %v", err)
		return nil, err
	}
	return prompts.CallLLMAPI[LLMResponsePhase2](
		s.httpClient,
		ctx,
		cfg.GetChatCompletionURL(),
		cfg.APIKey,
		cfg.ModelName,
		systemPrompt,
		prompt,
	)
}

func (s *GpuAnalysisService) buildPhase1Prompt(metrics *monitor.GpuAnalysisMetrics, processList string) string {
	data := map[string]any{
		"GpuUtilAvg":       fmt.Sprintf("%.2f", metrics.GpuUtilAvg),
		"GpuUtilStdDev":    fmt.Sprintf("%.2f", metrics.GpuUtilStdDev),
		"GpuMemUsedAvg":    fmt.Sprintf("%.2f", metrics.GpuMemUsedAvg),
		"GpuMemUsedStdDev": fmt.Sprintf("%.2f", metrics.GpuMemUsedStdDev),
		"ProcessList":      processList,
	}

	prompt, err := gpu_analysis.GetPhase1Prompt(data)
	if err != nil {
		klog.Errorf("Error building Phase 1 prompt: %v", err)
		return ""
	}
	return prompt
}

func (s *GpuAnalysisService) buildPhase2Prompt(
	metrics *monitor.GpuAnalysisMetrics,
	fullCommand,
	scriptContent,
	phase1Reason string,
) string {
	data := map[string]any{
		"Phase1Reason":  phase1Reason,
		"GpuUtilAvg":    fmt.Sprintf("%.2f", metrics.GpuUtilAvg),
		"GpuUtilStdDev": fmt.Sprintf("%.2f", metrics.GpuUtilStdDev), // 如果模板里有用到的话
		"GpuMemUsedAvg": fmt.Sprintf("%.2f", metrics.GpuMemUsedAvg),
		"FullCommand":   fullCommand,
		"ScriptContent": scriptContent,
	}

	prompt, err := gpu_analysis.GetPhase2Prompt(data)
	if err != nil {
		klog.Errorf("Failed to generate Phase 2 prompt from template: %v", err)
		return ""
	}

	return prompt
}

// --- Pod 内部操作辅助函数 ---

// 【修改】: findAndReadScriptInPod 现在接收 PID 而不是 nvidia-smi 的输出
func (s *GpuAnalysisService) findAndReadScriptInPod(
	ctx context.Context,
	namespace, podName,
	containerName string,
	pid int,
) (fullCommand, scriptContent string, err error) {
	// 1. 根据 PID 从 /proc 系统中获取完整的命令行
	cmdline, err := s.execCommandInPod(ctx, namespace, podName, containerName, []string{"cat", fmt.Sprintf("/proc/%d/cmdline", pid)})
	if err != nil {
		return "", "", fmt.Errorf("could not read cmdline for PID %d: %w", pid, err)
	}
	// /proc/cmdline 使用 null 字符分隔参数，替换为 空格
	fullCommand = strings.ReplaceAll(strings.TrimSpace(cmdline), "\x00", " ")

	// 2. 从完整命令行中提取脚本路径
	scriptPath := s.extractScriptPathFromCommand(fullCommand)
	if scriptPath == "" {
		return fullCommand, "", fmt.Errorf("no script file found in command: %s", fullCommand)
	}

	var absoluteScriptPath string
	if path.IsAbs(scriptPath) {
		// 如果 scriptPath 本身就是绝对路径，直接使用
		absoluteScriptPath = scriptPath
	} else {
		// 如果是相对路径，需要动态解析

		// 3.1. 【新增】动态获取进程的所有者用户名
		statCmd := []string{"stat", "-c", "%U", fmt.Sprintf("/proc/%d", pid)}
		usernameOutput, err := s.execCommandInPod(ctx, namespace, podName, containerName, statCmd)
		if err != nil {
			return fullCommand, "", fmt.Errorf("could not get owner username for PID %d using stat: %w", pid, err)
		}
		processUser := strings.TrimSpace(usernameOutput)
		if processUser == "" {
			return fullCommand, "", fmt.Errorf("got empty username for PID %d", pid)
		}

		// 3.2. 【修改】使用 runuser 和动态获取的用户名来获取 CWD
		// '--' 是一个好习惯，它告诉 runuser 后面的都是要执行的命令，即使它们以 '-' 开头
		runuserCmd := []string{"runuser", "-u", processUser, "--", "readlink", fmt.Sprintf("/proc/%d/cwd", pid)}
		cwd, err := s.execCommandInPod(ctx, namespace, podName, containerName, runuserCmd)
		if err != nil {
			return fullCommand, "", fmt.Errorf("could not get working directory for PID %d as user %s: %w", pid, processUser, err)
		}
		processCwd := strings.TrimSpace(cwd)

		// 3.3. 拼接成绝对路径
		absoluteScriptPath = path.Join(processCwd, scriptPath)
	}

	klog.Infof("Script path for PID %d resolved to: %s", pid, absoluteScriptPath)

	// 3. 读取脚本文件内容
	scriptContent, err = s.execCommandInPod(ctx, namespace, podName, containerName, []string{"cat", absoluteScriptPath})
	if err != nil {
		return fullCommand, "", fmt.Errorf("could not read script content from %s: %w", absoluteScriptPath, err)
	}

	// 限制脚本内容长度，避免过长
	if len(scriptContent) > maxScriptLength {
		scriptContent = scriptContent[:maxScriptLength] + "\n... (script truncated)"
	}

	return fullCommand, scriptContent, nil
}

func (s *GpuAnalysisService) extractScriptPathFromCommand(command string) string {
	// 使用正则表达式查找可能的脚本文件 (.py, .sh)
	//nolint:gocritic // Linter gives a strange false positive here.
	re := regexp.MustCompile(`([/\w\.-]+\.(?:py|sh))\b`)
	matches := re.FindStringSubmatch(command)
	if len(matches) > 1 {
		return matches[1]
	}
	return ""
}

func (s *GpuAnalysisService) execCommandInPod(
	ctx context.Context,
	namespace,
	podName,
	containerName string,
	command []string,
) (string, error) {
	req := s.kubeClient.CoreV1().RESTClient().Post().Resource("pods").Name(podName).Namespace(namespace).SubResource("exec")
	req.VersionedParams(&v1.PodExecOptions{
		Command: command, Container: containerName, Stdin: false, Stdout: true, Stderr: true, TTY: false,
	}, scheme.ParameterCodec)
	exec, err := remotecommand.NewSPDYExecutor(s.kubeConfig, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("failed to create SPDY executor: %w", err)
	}
	var stdout, stderr bytes.Buffer
	err = exec.StreamWithContext(ctx, remotecommand.StreamOptions{Stdout: &stdout, Stderr: &stderr})
	if err != nil {
		return "", fmt.Errorf("stream execution failed for command '%s', stderr: %s, error: %w", strings.Join(command, " "), stderr.String(), err)
	}
	if stderr.Len() > 0 {
		return stdout.String(), fmt.Errorf("command execution returned an error on stderr: %s", stderr.String())
	}
	return stdout.String(), nil
}

func (s *GpuAnalysisService) cleanupStaleAnalyses(ctx context.Context) {
	klog.Info("Starting cleanup of stale GPU analysis records...")
	ga := s.q.GpuAnalysis
	j := s.q.Job

	allAnalyses, err := ga.WithContext(ctx).Find()
	if err != nil {
		return
	}

	if len(allAnalyses) == 0 {
		return
	}

	jobIDs := make(map[uint]struct{})
	for _, analysis := range allAnalyses {
		jobIDs[analysis.JobID] = struct{}{}
	}

	jobIDSlice := make([]uint, 0, len(jobIDs))
	for id := range jobIDs {
		jobIDSlice = append(jobIDSlice, id)
	}

	// 3. 一次性查询所有相关的 Job
	// 使用 gen 的类型安全 Where 和 In
	jobs, err := j.WithContext(ctx).Where(j.ID.In(jobIDSlice...)).Find()
	if err != nil {
		return
	}

	// 4. 筛选出所有仍然有效的 Job ID
	eligibleJobIDs := make(map[uint]struct{})
	for _, job := range jobs {
		// 这里的 j 是 *model.Job 类型，可以直接调用我们之前定义的方法
		if IsEligibleForGpuAnalysis(job) {
			eligibleJobIDs[job.ID] = struct{}{}
		}
	}

	// 5. 找出需要删除的 GpuAnalysis 记录的 ID
	var analysisIDsToDelete []uint
	for _, analysis := range allAnalyses {
		_, isJobActive := eligibleJobIDs[analysis.JobID]
		// 如果 Job ID 不在有效列表中，则说明该分析记录是过时的
		if !isJobActive && analysis.ReviewStatus != model.ReviewStatusPending {
			analysisIDsToDelete = append(analysisIDsToDelete, analysis.ID)
		}
	}

	if len(analysisIDsToDelete) == 0 {
		return
	}

	// 6. 批量删除过时的记录
	// 使用 gen 的 Delete 方法
	_, err = ga.WithContext(ctx).Where(ga.ID.In(analysisIDsToDelete...)).Delete()
	if err != nil {
		return
	}
}

func IsEligibleForGpuAnalysis(j *model.Job) bool {
	if j == nil {
		return false
	}

	// 条件2: 检查作业是否正在运行
	if j.Status != "Running" {
		return false
	}

	// 条件3: 检查作业是否被锁定
	if !j.LockedTimestamp.IsZero() {
		return false
	}

	if time.Since(j.RunningTimestamp) < PodAnalysisMinAge {
		return false
	}

	// 条件1: 检查作业是否使用了 GPU
	hasGpu := false
	if j.Resources.Data() != nil {
		resources := j.Resources.Data()
		for resourceName, quantity := range resources {
			// 通用性检查，覆盖 nvidia.com/gpu, amd.com/gpu 等
			if strings.Contains(string(resourceName), "com") && quantity.Cmp(resource.MustParse("0")) > 0 {
				hasGpu = true
				break
			}
		}
	}

	return hasGpu
}

func (s *GpuAnalysisService) cleanOvertimeGpuAnalyses(ctx context.Context) {
	klog.Info("Starting cleanup of overtime GPU analysis records...")
	ga := s.q.GpuAnalysis
	cutoffTime := time.Now().Add(-GpuAnalysisMaxAge)
	// 超时且已review的记录可以删除
	_, err := ga.WithContext(ctx).Where(ga.CreatedAt.Lt(cutoffTime), ga.ReviewStatus.Neq(uint8(model.ReviewStatusPending))).Delete()
	if err != nil {
		klog.Errorf("Error cleaning up overtime GPU analysis records: %v", err)
		return
	}
}

func (s *GpuAnalysisService) makeSurePid(processList string, llmPid int) int {
	if llmPid > 0 {
		return llmPid
	}
	// 如果 LLM 没有返回 PID，则从 processList 中提取第一个 PID 作为备选
	return s.extractTopPidFromProcessList(processList)
}

func (s *GpuAnalysisService) extractTopPidFromProcessList(processList string) int {
	lines := strings.Split(strings.TrimSpace(processList), "\n")
	if len(lines) < 2 { // 第一行是表头，至少需要两行
		return 0
	}

	for i, line := range lines {
		if i == 0 {
			continue // 跳过表头
		}

		trimmedLine := strings.TrimSpace(line)
		if trimmedLine == "" {
			continue
		}

		// 将行拆分为列: [PID, CPU, MEM, USER, CMD...]
		fields := strings.Fields(trimmedLine)
		if len(fields) < psColumnNum {
			continue
		}

		pidStr := fields[0]
		// 拼接完整的命令行进行判断
		fullCmdInLine := strings.Join(fields[4:], " ")

		// --- 核心过滤逻辑 (在 Go 内存中执行，不依赖镜像工具) ---

		// 1. 排除诊断相关的关键词
		isDiagnostic := false
		diagnostics := []string{
			"ps -eo", "ps -ao", "ps aux",
			"grep", "head -n", "cat /proc",
		}
		for _, d := range diagnostics {
			if strings.Contains(fullCmdInLine, d) {
				isDiagnostic = true
				break
			}
		}
		if isDiagnostic {
			continue
		}

		pid, err := strconv.Atoi(pidStr)
		if err == nil && pid > 0 {
			return pid
		}
	}

	// 如果循环结束还没找到，那就实在没办法了，尝试返回 PID 1
	// 因为大多数容器的业务进程就是 PID 1
	return 1
}
