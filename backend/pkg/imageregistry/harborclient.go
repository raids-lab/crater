package imageregistry

import (
	"fmt"

	haborapiv2 "github.com/mittwald/goharbor-client/v5/apiv2"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/pkg/config"
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
	harborConfig := config.GetConfig().ImageRegistry
	HarborAPIServer := fmt.Sprintf("https://%s/api/", harborConfig.Server)
	restClient, err := haborapiv2.NewRESTClientForHost(
		HarborAPIServer,
		harborConfig.Admin,
		harborConfig.AdminPassword,
		nil,
	)
	if err != nil {
		klog.Errorf("establish harbor client failed, err: %+v", err)
	}
	authInfo := AuthInfo{
		RegistryServer:  harborConfig.Server,
		RegistryUser:    harborConfig.User,
		RegistryPass:    harborConfig.Password,
		RegistryProject: harborConfig.Project,
	}
	return HarborClient{*restClient, authInfo}
}
