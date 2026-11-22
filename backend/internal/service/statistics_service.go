package service

import (
	"context"
	"time"

	"github.com/raids-lab/crater/dao/query"
	"github.com/raids-lab/crater/internal/bizerr"
	"github.com/raids-lab/crater/internal/payload"
)

// StatisticsService 提供资源日统计及排名查询。
type StatisticsService struct {
	q *query.Query
}

// NewStatisticsService 创建统计服务。
func NewStatisticsService(q *query.Query) *StatisticsService {
	return &StatisticsService{q: q}
}

func statisticsFilter(req *payload.StatisticsReq) *Filter {
	return &Filter{Start: req.StartTime.UTC(), End: req.EndTime.UTC(),
		Scope: string(req.Scope), Step: string(req.Step), TargetID: req.TargetID}
}

// GetResourceStatistics 查询资源用量和统计进度。
func (s *StatisticsService) GetResourceStatistics(
	ctx context.Context, req *payload.StatisticsReq, allocatedOnly bool,
) (*payload.StatisticsResp, error) {
	filter := statisticsFilter(req)
	filter.allocatedOnly = allocatedOnly
	response, err := s.overview(ctx, filter)
	if err != nil {
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "read daily resource statistics")
	}
	return response, nil
}

// GetRankings 查询资源用量或利用率排名。
func (s *StatisticsService) GetRankings(ctx context.Context, req *payload.StatisticsRankingReq) (*payload.StatisticsRankingResp, error) {
	response, err := s.rankings(ctx, &RankingFilter{
		Filter: *statisticsFilter(&req.StatisticsReq), Dimension: req.Dimension, Metric: req.Metric, Order: req.Order,
		ResourceName: req.ResourceName, ResourceType: req.ResourceType, Page: req.Page, PageSize: req.PageSize,
	})
	if err != nil {
		if _, ok := bizerr.FromError(err); ok {
			return nil, err
		}
		return nil, bizerr.Internal.DatabaseError.Wrap(err, "read daily resource rankings")
	}
	return response, nil
}

// RunDailyAggregation 是巡检任务的日统计入口。
func (s *StatisticsService) RunDailyAggregation(ctx context.Context) (any, error) {
	return s.Refresh(ctx, time.Now())
}
