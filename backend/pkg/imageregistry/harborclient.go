package imageregistry

import (
	"fmt"

	haborapiv2 "github.com/mittwald/goharbor-client/v5/apiv2"

	"github.com/raids-lab/crater/pkg/config"
	"github.com/raids-lab/crater/pkg/logutils"
)

type AuthInfo struct {
	RegistryServer  string
	RegistryUser    string
	RegistryPass    string
	RegistryProject string
}

type HarborClient struct {
	haborapiv2.RESTClient
	AuthInfo
}

func NewHarborClient() HarborClient {
	harborConfig := config.GetConfig().ACT.Image
	HarborAPIServer := fmt.Sprintf("https://%s/api/", harborConfig.RegistryServer)
	restClient, err := haborapiv2.NewRESTClientForHost(HarborAPIServer, harborConfig.RegistryAdmin, harborConfig.RegistryAdminPass, nil)
	if err != nil {
		logutils.Log.Errorf("establish harbor client failed, err: %+v", err)
	}
	authInfo := AuthInfo{
		RegistryServer:  harborConfig.RegistryServer,
		RegistryUser:    harborConfig.RegistryUser,
		RegistryPass:    harborConfig.RegistryPass,
		RegistryProject: harborConfig.RegistryProject,
	}
	return HarborClient{*restClient, authInfo}
}
