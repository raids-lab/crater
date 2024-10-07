package handler

import (
	"github.com/gin-gonic/gin"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Manager is the interface that wraps the basic methods for a handler manager.
type Manager interface {
	GetName() string
	RegisterPublic(group *gin.RouterGroup)
	RegisterProtected(group *gin.RouterGroup)
	RegisterAdmin(group *gin.RouterGroup)
}

// RegisterConfig is a struct that holds the configuration for a Manager.
type RegisterConfig struct {
	// Client is the controller-runtime client.
	Client client.Client

	// KubeClient is the kubernetes client.
	KubeClient kubernetes.Interface

	// PrometheusClient is the prometheus client.
}

// Registers is a slice of Manager Init functions.
// Each Manager should register itself by appending its Init function to this slice.
var Registers = []func(config RegisterConfig) Manager{}
