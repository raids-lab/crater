package internal

import (
	"github.com/raids-lab/crater/internal/handler"
	_ "github.com/raids-lab/crater/internal/handler/aijob"
	_ "github.com/raids-lab/crater/internal/handler/image"
	_ "github.com/raids-lab/crater/internal/handler/operations"
	_ "github.com/raids-lab/crater/internal/handler/spjob"
	_ "github.com/raids-lab/crater/internal/handler/tool"
	_ "github.com/raids-lab/crater/internal/handler/vcjob"
	"github.com/raids-lab/crater/pkg/logutils"
)

// registerManagers registers all the managers.
func registerManagers(config *handler.RegisterConfig) []handler.Manager {
	var managers []handler.Manager
	for _, register := range handler.Registers {
		manager := register(config)
		managers = append(managers, manager)
		logutils.Log.Infof("Registered manager: %s", manager.GetName())
	}
	return managers
}
