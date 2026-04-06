package patrol

import (
	"context"
	"fmt"
)

// RunTriggerAdminOpsReport is the cron entry for the scheduled admin ops report.
func RunTriggerAdminOpsReport(
	ctx context.Context,
	clients *Clients,
	req TriggerAdminOpsReportRequest,
) (any, error) {
	if clients.AdminOpsService == nil {
		return nil, fmt.Errorf("admin ops report service is not initialized in patrol clients")
	}

	result, err := clients.AdminOpsService.TriggerAdminOpsReport(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to trigger admin ops report patrol: %w", err)
	}
	return result, nil
}
