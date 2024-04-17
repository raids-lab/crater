package crclient

import (
	"fmt"

	haborapiv2 "github.com/mittwald/goharbor-client/v5/apiv2"
	"github.com/raids-lab/crater/pkg/logutils"
)

const (
	RegistryServer  = "***REMOVED***"
	RegistryUser    = "***REMOVED***"
	RegistryPass    = "***REMOVED***" //nolint:gosec // 暂时硬编码这四个参数
	RegistryProject = "crater-images"
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
	HarborAPIServer := fmt.Sprintf("https://%s/api/", RegistryServer)
	restClient, err := haborapiv2.NewRESTClientForHost(HarborAPIServer, RegistryUser, RegistryPass, nil)
	if err != nil {
		logutils.Log.Errorf("establish harbor client failed, err: %+v", err)
	}
	authInfo := AuthInfo{
		RegistryServer:  RegistryServer,
		RegistryUser:    RegistryUser,
		RegistryPass:    RegistryPass,
		RegistryProject: RegistryProject,
	}
	return HarborClient{*restClient, authInfo}
}
