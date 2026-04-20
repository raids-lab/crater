package patrol

import (
	"context"
	"fmt"
)

// RunTriggerBillingBaseLoop is the patrol entry for one-shot billing base loop.
func RunTriggerBillingBaseLoop(ctx context.Context, clients *Clients) (any, error) {
	if clients.BillingService == nil {
		return nil, fmt.Errorf("billing service is not initialized in patrol clients")
	}
	return clients.BillingService.RunBaseLoopOnce(ctx)
}
