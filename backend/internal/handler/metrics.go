package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/raids-lab/crater/dao/model"
	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/resputil"
)

type MetricsMgr struct {
	name string
}

func NewMetricsMgr(_ *RegisterConfig) Manager {
	return &MetricsMgr{
		name: "metrics",
	}
}

func (mgr *MetricsMgr) GetName() string { return mgr.name }

func (mgr *MetricsMgr) RegisterPublic(metrics *gin.RouterGroup) {
	metrics.GET("", mgr.GetMetrics)
}

func (mgr *MetricsMgr) RegisterProtected(_ *gin.RouterGroup) {}

func (mgr *MetricsMgr) RegisterAdmin(_ *gin.RouterGroup) {}

// 声明一个自定义的注册表
var registry *prometheus.Registry

// 声明一个prom HTTP Handler
var promHTTPHandler http.Handler

// 已完成的 Job 仪表盘
var completedJobsGauge = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "completed_jobs_total",
		Help: "Total number of completed jobs",
	},
)

// 正在运行的 Job 仪表盘
var runningJobsGauge = prometheus.NewGauge(
	prometheus.GaugeOpts{
		Name: "running_jobs_total",
		Help: "Total number of running jobs",
	},
)

//nolint:gochecknoinits // This is the standard way to register a gin handler.
func init() {
	Registers = append(Registers, NewMetricsMgr)
	registry = prometheus.NewRegistry()
	promHTTPHandler = promhttp.HandlerFor(registry, promhttp.HandlerOpts{Registry: registry})
	registry.MustRegister(completedJobsGauge)
	registry.MustRegister(runningJobsGauge)
}

// GetMetrics godoc
// @Summary 获取系统中每种Status的Job的数量
// @Description 返回Prometheus能够识别的信息
// @Tags Metrics
// @Accept json
// @Produce json
// @Security Bearer
// @Success 200 {array} resputil.Response[any] "成功返回"
// @Failure 400 {object} resputil.Response[any] "请求参数错误"
// @Failure 500 {object} resputil.Response[any] "其他错误"
// @Router /metrics [get]
func (mgr *MetricsMgr) GetMetrics(c *gin.Context) {
	j := query.Job
	jobs, err := j.WithContext(c).Find()
	if err != nil {
		resputil.Error(c, err.Error(), resputil.NotSpecified)
		return
	}
	getJobNum(jobs)
	// 暴露自定义指标
	promHTTPHandler.ServeHTTP(c.Writer, c.Request)
}

func getJobNum(jobs []*model.Job) {
	completedCount, runningCount := 0, 0
	for i := range jobs {
		switch jobs[i].Status {
		case "Completed":
			completedCount++
		case "Running":
			runningCount++
		}
	}
	completedJobsGauge.Set(float64(completedCount))
	runningJobsGauge.Set(float64(runningCount))
}
