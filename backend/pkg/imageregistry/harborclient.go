package imageregistry

import (
	"fmt"

	haborapiv2 "github.com/mittwald/goharbor-client/v5/apiv2"
	"k8s.io/klog/v2"

	"github.com/raids-lab/crater/pkg/config"
)

type AuthInfo struct {
	RegistryServer string
}

type HarborClient struct {
	haborapiv2.RESTClient
	AuthInfo
}

func NewHarborClient() HarborClient {
	harborConfig := config.GetConfig().Registry.Harbor
	HarborAPIServer := fmt.Sprintf("https://%s/api/", harborConfig.Server)
	restClient, err := haborapiv2.NewRESTClientForHost(
		HarborAPIServer,
		harborConfig.User,
		harborConfig.Password,
		nil,
	)
	if err != nil {
		klog.Errorf("establish harbor client failed, err: %+v", err)
	}
	authInfo := AuthInfo{
		RegistryServer: harborConfig.Server,
	}
	return HarborClient{*restClient, authInfo}
}
