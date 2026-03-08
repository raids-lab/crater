package config

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"
	"sigs.k8s.io/yaml"
)

type Config struct {
	// EnableLeaderElection enables leader election for controller manager to ensure high availability.
	// Optional: Defaults to false if not specified.
	EnableLeaderElection bool `json:"enableLeaderElection"`

	// Host defines the domain name or IP address that the server will bind to.
	// Required: Must be specified for the server to start.
	Host string `json:"host"`

	// Port defines the network port that the server endpoint will listen on.
	// Required: Must be specified for the server to start.
	Port string `json:"port"`

	// Namespaces contains Kubernetes namespace configurations for different resources.
	// Required: Both Job and Image namespaces must be specified.
	Namespaces struct {
		// Job namespace where job resources will be created and managed.
		// Required: Must be a valid Kubernetes namespace.
		Job string `json:"job"`

		// Image namespace where image resources will be stored and managed.
		// Required: Must be a valid Kubernetes namespace.
		Image string `json:"image"`
	} `json:"namespaces"`

	// PrometheusAPI specifies the endpoint URL for Prometheus API used for metrics and monitoring.
	// Optional: If not specified, Prometheus integration will be disabled.
	PrometheusAPI string `json:"prometheusAPI"`

	// Postgres contains PostgreSQL database connection configuration.
	// Required: All fields must be specified for database connectivity.
	Postgres struct {
		// Host is the PostgreSQL server hostname or IP address.
		// Required: Must be reachable from the application.
		Host string `json:"host"`

		// Port is the PostgreSQL server port number.
		// Required: Typically 5432 for PostgreSQL.
		Port string `json:"port"`

		// DBName is the name of the database to connect to.
		// Required: Database must exist and be accessible.
		DBName string `json:"dbname"`

		// User is the database username for authentication.
		// Required: User must have appropriate permissions.
		User string `json:"user"`

		// Password is the database password for authentication.
		// Required: Must match the specified user's password.
		Password string `json:"password"`

		// SSLMode specifies the SSL/TLS mode for database connection.
		// Optional: Defaults to "disable" if not specified.
		SSLMode string `json:"sslmode"`

		// TimeZone specifies the time zone for database connections.
		// Optional: Defaults to system time zone if not specified.
		TimeZone string `json:"TimeZone"`
	} `json:"postgres"`

	// Storage contains persistent volume claim and path prefix configurations.
	// Required: All PVC names and prefix paths must be specified.
	Storage struct {
		PVC struct {
			// ReadWriteMany is the name of the ReadWriteMany Persistent Volume Claim for shared storage.
			// Required: PVC must exist in the cluster with ReadWriteMany access mode.
			ReadWriteMany string `json:"readWriteMany"`

			// ReadOnlyMany is the name of the ReadOnlyMany Persistent Volume Claim for datasets and models.
			// It should be a link to the same underlying storage as ReadWriteMany.
			// Optional: If not specified, datasets and models will be mounted as read-write.
			ReadOnlyMany *string `json:"readOnlyMany,omitempty"`
		} `json:"pvc"`

		// Prefix contains path prefixes for different types of storage locations.
		// Required: All prefix paths must be specified.
		Prefix struct {
			// User prefix for user-specific storage paths.
			// Required: Must be a valid path within the storage system.
			User string `json:"user"`

			// Account prefix for account-related storage paths.
			// Required: Must be a valid path within the storage system.
			Account string `json:"account"`

			// Public prefix for publicly accessible storage paths.
			// Required: Must be a valid path within the storage system.
			Public string `json:"public"`
		} `json:"prefix"`
	} `json:"storage"`

	// Secrets contains Kubernetes secret names for various security components.
	// Required: All secret names must correspond to existing Kubernetes secrets.
	Secrets struct {
		// TLSSecretName is the name of the Kubernetes secret containing TLS certificates for HTTPS.
		// Required: Secret must contain 'tls.crt' and 'tls.key' keys.
		TLSSecretName string `json:"tlsSecretName"`

		// TLSForwardSecretName is the name of the Kubernetes secret for TLS forwarding configuration.
		// Required: Secret must contain appropriate forwarding certificates.
		TLSForwardSecretName string `json:"tlsForwardSecretName"`

		// ImagePullSecretName is the name of the Kubernetes secret for pulling container images from private registries.
		// Optional: If not specified, no image pull secret will be used.
		ImagePullSecretName string `json:"imagePullSecretName"`
	}

	// ModelDownload contains configurations for model download functionality.
	// Optional: If not specified, default values will be used.
	ModelDownload struct {
		// Image is the container image used for model download jobs.
		// Optional: Defaults to "crater-harbor.act.buaa.edu.cn/docker.io/python:3.11-slim" if not specified.
		Image string `json:"image"`
	} `json:"modelDownload"`

	// Registry contains container registry configuration for image storage and building.
	// Optional: If Enable is false, registry functionality will be disabled.
	Registry struct {
		// Enable toggles container registry integration.
		// Optional: Defaults to false if not specified.
		Enable bool `json:"enable"`

		// Harbor contains configuration for Harbor container registry integration.
		// Required if Registry.Enable is true: All Harbor fields must be specified.
		Harbor struct {
			// Server is the Harbor registry server URL.
			// Required: Must be a valid Harbor instance URL.
			Server string `json:"server"`

			// User is the username for Harbor authentication.
			// Required: User must have appropriate permissions in Harbor.
			User string `json:"user"`

			// Password is the password for Harbor authentication.
			// Required: Must match the specified user's password.
			Password string `json:"password"`
		} `json:"harbor"`

		// BuildTools contains configuration for container image building tools and proxies.
		// Required if Registry.Enable is true.
		BuildTools struct {
			// ProxyConfig contains HTTP proxy settings for build environments.
			// Optional: If not specified, no proxy will be configured for builds.
			ProxyConfig struct {
				// HTTPSProxy is the HTTPS proxy URL for build environments.
				// Optional: If not specified, HTTPS traffic will not be proxied.
				HTTPSProxy string `json:"httpsProxy"`

				// HTTPProxy is the HTTP proxy URL for build environments.
				// Optional: If not specified, HTTP traffic will not be proxied.
				HTTPProxy string `json:"httpProxy"`

				// NoProxy is a comma-separated list of domains that should not be proxied.
				// Optional: If not specified, all traffic will go through the proxy.
				NoProxy string `json:"noProxy"`
			} `json:"proxyConfig"`

			// Images contains container image references for various build tools.
			// Required if Registry.Enable is true.
			Images struct {
				// Buildx image for Docker Buildx multi-platform builds.
				// Required if Registry.Enable is true.
				Buildx string `json:"buildx"`

				// Nerdctl image for containerd-based builds.
				// Required if Registry.Enable is true.
				Nerdctl string `json:"nerdctl"`

				// Envd image for environment-based development builds.
				// Required if Registry.Enable is true.
				Envd string `json:"envd"`
			} `json:"images"`
		} `json:"buildTools"`
	} `json:"registry"`

	// SMTP contains configuration for email notifications via SMTP.
	// Optional: If Enable is false, email notifications will be disabled.
	SMTP struct {
		// Enable toggles SMTP email functionality.
		// Optional: Defaults to false if not specified.
		Enable bool `json:"enable"`

		// Host is the SMTP server hostname or IP address.
		// Required if Enable is true: Must be a valid SMTP server.
		Host string `json:"host"`

		// Port is the SMTP server port number.
		// Required if Enable is true: Typically 25, 465, or 587.
		Port string `json:"port"`

		// User is the username for SMTP authentication.
		// Required if Enable is true: Must be a valid SMTP user.
		User string `json:"user"`

		// Password is the password for SMTP authentication.
		// Required if Enable is true: Must match the specified user's password.
		Password string `json:"password"`

		// Notify is the default email address for system notifications.
		// Required if Enable is true: Must be a valid email address.
		Notify string `json:"notify"`
	} `json:"smtp"`

	// Auth contains configuration for various authentication methods and tokens.
	Auth struct {
		// Token contains authentication token configuration for JWT-based authentication.
		Token struct {
			// AccessTokenSecret is the secret key used to sign JWT access tokens.
			AccessTokenSecret string `json:"accessTokenSecret"`
			// RefreshTokenSecret is the secret key used to sign JWT refresh tokens.
			RefreshTokenSecret string `json:"refreshTokenSecret"`
		} `json:"token"`

		// LDAP authentication settings.
		LDAP struct {
			// Enable toggles LDAP authentication.
			// Optional: Defaults to false if not specified.
			Enable bool `json:"enable"`

			// Server connection settings.
			Server struct {
				// Address is the LDAP server address (host:port).
				Address string `json:"address"`
				// BaseDN is the search base.
				BaseDN string `json:"baseDN"`
				// BindDN is the account for initial bind.
				BindDN string `json:"bindDN"`
				// BindPassword is the password for initial bind.
				BindPassword string `json:"bindPassword"`
			} `json:"server"`

			// AttributeMapping defines how LDAP attributes map to User fields.
			AttributeMapping struct {
				// Username is the LDAP attribute for login name.
				Username string `json:"username"`
				// DisplayName maps to user's nickname.
				DisplayName string `json:"displayName"`
				// Email maps to user's email.
				Email string `json:"email"`
				// Teacher maps to user's teacher/advisor.
				Teacher string `json:"teacher"`
				// Group maps to user's group.
				Group string `json:"group"`
				// Phone maps to user's phone.
				Phone string `json:"phone"`
				// ExpiredAt maps to user's account expiration.
				ExpiredAt string `json:"expiredAt"`
			} `json:"attributeMapping"`

			// UID contains settings for UID/GID acquisition.
			UID struct {
				// Source defines where to get UID/GID ("none", "ldap", "external").
				Source string `json:"source"`
				// LDAPAttribute defines attribute names when source is "ldap".
				LDAPAttribute struct {
					UID string `json:"uid"`
					GID string `json:"gid"`
				} `json:"ldapAttribute"`
				// ExternalService defines service details when source is "external".
				ExternalService struct {
					URL     string `json:"url"`
					Timeout int    `json:"timeout"`
				} `json:"externalService"`
			} `json:"uid"`
		} `json:"ldap"`

		// Normal contains settings for local database-based authentication.
		Normal struct {
			// AllowRegister allows users to sign up via the platform.
			AllowRegister bool `json:"allowRegister"`
			// AllowLogin allows users to login using local credentials when LDAP is enabled.
			AllowLogin bool `json:"allowLogin"`
		} `json:"normal"`
	} `json:"auth"`

	// SchedulerPlugins contains configuration for Kubernetes scheduler plugin integrations.
	// Optional: Individual plugins can be enabled/disabled independently.
	SchedulerPlugins struct {
		// EMIAS contains configuration for AI Job scheduler plugin.
		// Optional: If Enable is false, EMIAS plugin will be disabled.
		EMIAS struct {
			// Enable toggles the EMIAS scheduler plugin.
			// Optional: Defaults to false if not specified.
			Enable bool `json:"enable"`

			// EnableProfiling toggles profiling capabilities for the EMIAS plugin.
			// Optional: Defaults to false if not specified.
			EnableProfiling bool `json:"enableProfiling"`

			// ProfilingTimeout specifies the timeout in seconds for profiling operations.
			// Optional: Defaults to 30 seconds if not specified.
			ProfilingTimeout int `json:"profilingTimeout"`
		} `json:"aijob"`

		// SEACS contains configuration for SP Job scheduler plugin with prediction capabilities.
		// Optional: If Enable is false, SEACS plugin will be disabled.
		SEACS struct {
			// Enable toggles the SEACS scheduler plugin.
			// Optional: Defaults to false if not specified.
			Enable bool `json:"enable"`

			// PredictionServiceAddress is the endpoint URL for the prediction service used by SEACS.
			// Required if Enable is true: Must be a valid prediction service endpoint.
			PredictionServiceAddress string `json:"predictionServiceAddress"`
		} `json:"spjob"`
	} `json:"schedulerPlugins"`
}

// ValidateConfig validates the configuration structure and checks for required fields
//
//nolint:gocyclo // This is long but simple.
func (c *Config) ValidateConfig() error {
	var errors []string

	// Validate basic required fields
	if c.Host == "" {
		errors = append(errors, "host is required")
	}
	if c.Port == "" {
		errors = append(errors, "port is required")
	}

	// Validate namespaces
	if c.Namespaces.Job == "" {
		errors = append(errors, "namespaces.job is required")
	}
	if c.Namespaces.Image == "" {
		errors = append(errors, "namespaces.image is required")
	}

	// Validate auth configuration
	if c.Auth.Token.AccessTokenSecret == "" {
		errors = append(errors, "auth.token.accessTokenSecret is required")
	}
	if c.Auth.Token.RefreshTokenSecret == "" {
		errors = append(errors, "auth.token.refreshTokenSecret is required")
	}

	// Validate postgres configuration
	if c.Postgres.Host == "" {
		errors = append(errors, "postgres.host is required")
	}
	if c.Postgres.Port == "" {
		errors = append(errors, "postgres.port is required")
	}
	if c.Postgres.DBName == "" {
		errors = append(errors, "postgres.dbname is required")
	}
	if c.Postgres.User == "" {
		errors = append(errors, "postgres.user is required")
	}
	if c.Postgres.Password == "" {
		errors = append(errors, "postgres.password is required")
	}

	// Validate storage configuration
	if c.Storage.PVC.ReadWriteMany == "" {
		errors = append(errors, "storage.pvc.rwxpvcName is required")
	}
	if c.Storage.Prefix.User == "" {
		errors = append(errors, "storage.prefix.user is required")
	}
	if c.Storage.Prefix.Account == "" {
		errors = append(errors, "storage.prefix.account is required")
	}
	if c.Storage.Prefix.Public == "" {
		errors = append(errors, "storage.prefix.public is required")
	}

	// Validate secrets configuration
	if c.Secrets.TLSSecretName == "" {
		errors = append(errors, "secrets.tlsSecretName is required")
	}
	if c.Secrets.TLSForwardSecretName == "" {
		errors = append(errors, "secrets.tlsForwardSecretName is required")
	}

	// Validate conditional configurations
	if c.Registry.Enable {
		if c.Registry.Harbor.Server == "" {
			errors = append(errors, "registry.harbor.server is required when registry is enabled")
		}
		if c.Registry.Harbor.User == "" {
			errors = append(errors, "registry.harbor.user is required when registry is enabled")
		}
		if c.Registry.Harbor.Password == "" {
			errors = append(errors, "registry.harbor.password is required when registry is enabled")
		}
		if c.Registry.BuildTools.Images.Buildx == "" {
			errors = append(errors, "registry.buildTools.images.buildx is required when registry is enabled")
		}
		if c.Registry.BuildTools.Images.Nerdctl == "" {
			errors = append(errors, "registry.buildTools.images.nerdctl is required when registry is enabled")
		}
		if c.Registry.BuildTools.Images.Envd == "" {
			errors = append(errors, "registry.buildTools.images.envd is required when registry is enabled")
		}
	}

	if c.SMTP.Enable {
		if c.SMTP.Host == "" {
			errors = append(errors, "smtp.host is required when smtp is enabled")
		}
		if c.SMTP.Port == "" {
			errors = append(errors, "smtp.port is required when smtp is enabled")
		}
		if c.SMTP.User == "" {
			errors = append(errors, "smtp.user is required when smtp is enabled")
		}
		if c.SMTP.Password == "" {
			errors = append(errors, "smtp.password is required when smtp is enabled")
		}
		if c.SMTP.Notify == "" {
			errors = append(errors, "smtp.notify is required when smtp is enabled")
		}
	}

	if c.Auth.LDAP.Enable {
		if c.Auth.LDAP.Server.BindDN == "" {
			errors = append(errors, "auth.ldap.server.bindDN is required when ldap is enabled")
		}
		if c.Auth.LDAP.Server.BindPassword == "" {
			errors = append(errors, "auth.ldap.server.bindPassword is required when ldap is enabled")
		}
		if c.Auth.LDAP.Server.Address == "" {
			errors = append(errors, "auth.ldap.server.address is required when ldap is enabled")
		}
		if c.Auth.LDAP.Server.BaseDN == "" {
			errors = append(errors, "auth.ldap.server.baseDN is required when ldap is enabled")
		}
		if c.Auth.LDAP.AttributeMapping.Username == "" {
			errors = append(errors, "auth.ldap.attributeMapping.username is required when ldap is enabled")
		}
		if c.Auth.LDAP.AttributeMapping.DisplayName == "" {
			errors = append(errors, "auth.ldap.attributeMapping.displayName is required when ldap is enabled "+
				"(can be same as username if no nickname field exists)")
		}
	}

	// Validate authentication logic: at least one login method must be enabled
	if !c.Auth.LDAP.Enable && !c.Auth.Normal.AllowLogin {
		errors = append(errors, "at least one authentication method (LDAP or Normal Login) must be enabled")
	}

	// Validate authentication logic: normal register requires normal login
	if !c.Auth.Normal.AllowLogin && c.Auth.Normal.AllowRegister {
		errors = append(errors, "normal registration cannot be enabled when normal login is disabled")
	}

	if c.Auth.LDAP.Enable {
		switch c.Auth.LDAP.UID.Source {
		case "ldap":
			if c.Auth.LDAP.UID.LDAPAttribute.UID == "" {
				errors = append(errors, "auth.ldap.uid.ldapAttribute.uid is required when uid.source is 'ldap'")
			}
			if c.Auth.LDAP.UID.LDAPAttribute.GID == "" {
				errors = append(errors, "auth.ldap.uid.ldapAttribute.gid is required when uid.source is 'ldap'")
			}
		case "external":
			if c.Auth.LDAP.UID.ExternalService.URL == "" {
				errors = append(errors, "auth.ldap.uid.externalService.url is required when uid.source is 'external'")
			}
		case "none", "default", "":
			// OK
		default:
			errors = append(errors, fmt.Sprintf("invalid auth.ldap.uid.source: %s", c.Auth.LDAP.UID.Source))
		}
	}

	if c.SchedulerPlugins.SEACS.Enable {
		if c.SchedulerPlugins.SEACS.PredictionServiceAddress == "" {
			errors = append(errors, "schedulerPlugins.spjob.predictionServiceAddress is required when SEACS is enabled")
		}
	}

	// Return validation errors if any
	if len(errors) > 0 {
		return fmt.Errorf("configuration validation failed:\n- %s", strings.Join(errors, "\n- "))
	}

	return nil
}

// PrintConfig prints the configuration in a formatted and readable way, masking sensitive information
func (c *Config) PrintConfig() {
	klog.Info("=== Configuration Summary ===")

	// Basic configuration
	klog.Infof("Server: %s:%s", c.Host, c.Port)
	klog.Infof("Leader Election: %t", c.EnableLeaderElection)

	// Namespaces
	klog.Infof("Namespaces - Job: %s, Image: %s", c.Namespaces.Job, c.Namespaces.Image)

	// Prometheus
	if c.PrometheusAPI != "" {
		klog.Infof("Prometheus API: %s", c.PrometheusAPI)
	} else {
		klog.Info("Prometheus API: <not configured>")
	}

	// Database
	klog.Infof("Database: %s@%s:%s/%s (SSL: %s, TZ: %s)",
		c.Postgres.User, c.Postgres.Host, c.Postgres.Port, c.Postgres.DBName,
		c.Postgres.SSLMode, c.Postgres.TimeZone)

	// Storage
	klog.Infof("Storage PVC: RWX=%s", c.Storage.PVC.ReadWriteMany)
	if c.Storage.PVC.ReadOnlyMany != nil {
		klog.Infof("Storage PVC: ROX=%s", *c.Storage.PVC.ReadOnlyMany)
	}
	klog.Infof("Storage Prefixes: User=%s, Account=%s, Public=%s",
		c.Storage.Prefix.User, c.Storage.Prefix.Account, c.Storage.Prefix.Public)

	// Model Download
	if c.ModelDownload.Image != "" {
		klog.Infof("Model Download Image: %s", c.ModelDownload.Image)
	} else {
		klog.Info("Model Download Image: <default: crater-harbor.act.buaa.edu.cn/crater/base/python:3.11-slim>")
	}

	// Secrets
	klog.Infof("TLS Secrets: %s, %s", c.Secrets.TLSSecretName, c.Secrets.TLSForwardSecretName)
	if c.Secrets.ImagePullSecretName != "" {
		klog.Infof("Image Pull Secret: %s", c.Secrets.ImagePullSecretName)
	}

	// Registry
	if c.Registry.Enable {
		klog.Infof("Registry: Enabled (Harbor: %s, User: %s)", c.Registry.Harbor.Server, c.Registry.Harbor.User)
		klog.Infof("Build Tools: Buildx=%s, Nerdctl=%s, Envd=%s",
			c.Registry.BuildTools.Images.Buildx, c.Registry.BuildTools.Images.Nerdctl, c.Registry.BuildTools.Images.Envd)
		if c.Registry.BuildTools.ProxyConfig.HTTPProxy != "" || c.Registry.BuildTools.ProxyConfig.HTTPSProxy != "" {
			klog.Infof("Build Proxy: HTTP=%s, HTTPS=%s, NoProxy=%s",
				c.Registry.BuildTools.ProxyConfig.HTTPProxy,
				c.Registry.BuildTools.ProxyConfig.HTTPSProxy,
				c.Registry.BuildTools.ProxyConfig.NoProxy)
		}
	} else {
		klog.Info("Registry: Disabled")
	}

	// SMTP
	if c.SMTP.Enable {
		klog.Infof("SMTP: Enabled (%s:%s, User: %s, Notify: %s)",
			c.SMTP.Host, c.SMTP.Port, c.SMTP.User, c.SMTP.Notify)
	} else {
		klog.Info("SMTP: Disabled")
	}

	// Authentication
	if c.Auth.LDAP.Enable {
		klog.Infof("LDAP Authentication: Enabled (Server: %s, BaseDN: %s)",
			c.Auth.LDAP.Server.Address, c.Auth.LDAP.Server.BaseDN)
		klog.Infof("LDAP UID Source: %s", c.Auth.LDAP.UID.Source)
		if c.Auth.LDAP.UID.Source == "external" {
			klog.Infof("LDAP External UID Service: %s", c.Auth.LDAP.UID.ExternalService.URL)
		}
	} else {
		klog.Info("LDAP Authentication: Disabled")
	}

	// Scheduler Plugins
	var enabledPlugins []string
	if c.SchedulerPlugins.EMIAS.Enable {
		pluginInfo := fmt.Sprintf("EMIAS(profiling: %t", c.SchedulerPlugins.EMIAS.EnableProfiling)
		if c.SchedulerPlugins.EMIAS.ProfilingTimeout > 0 {
			pluginInfo += fmt.Sprintf(", timeout: %ds", c.SchedulerPlugins.EMIAS.ProfilingTimeout)
		}
		pluginInfo += ")"
		enabledPlugins = append(enabledPlugins, pluginInfo)
	}
	if c.SchedulerPlugins.SEACS.Enable {
		pluginInfo := fmt.Sprintf("SEACS(prediction: %s)", c.SchedulerPlugins.SEACS.PredictionServiceAddress)
		enabledPlugins = append(enabledPlugins, pluginInfo)
	}

	if len(enabledPlugins) > 0 {
		klog.Infof("Scheduler Plugins: %s", strings.Join(enabledPlugins, ", "))
	} else {
		klog.Info("Scheduler Plugins: None enabled")
	}

	klog.Info("=== End Configuration Summary ===")
}

var (
	once   sync.Once
	config *Config
)

func GetConfig() *Config {
	once.Do(func() {
		config = initConfig()
	})
	return config
}

func IsDebugMode() bool {
	return gin.Mode() == gin.DebugMode
}

// initConfig initializes the configuration by reading the configuration file.
// If the environment is set to debug, it reads the debug-config.yaml file.
// Otherwise, it reads the config.yaml file from ConfigMap.
// It returns a pointer to the Config struct and an error if any occurred.
func initConfig() *Config {
	// 读取配置文件
	config := &Config{}
	var configPath string
	if IsDebugMode() {
		if os.Getenv("CRATER_DEBUG_CONFIG_PATH") != "" {
			configPath = os.Getenv("CRATER_DEBUG_CONFIG_PATH")
		} else {
			configPath = "./etc/debug-config.yaml"
		}
	} else {
		configPath = "/etc/config/config.yaml"
	}
	klog.Infof("Loading configuration from: %s", configPath)

	err := readConfig(configPath, config)
	if err != nil {
		klog.Fatalf("Failed to read config file: %v", err)
	}

	// Validate configuration
	if err := config.ValidateConfig(); err != nil {
		klog.Fatalf("Configuration validation failed: %v", err)
	}

	// Print configuration summary
	config.PrintConfig()

	klog.Info("Configuration loaded and validated successfully")
	return config
}

func readConfig(filePath string, config *Config) error {
	// 读取 YAML 配置文件
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}
	// 解析 YAML 数据到结构体
	err = yaml.Unmarshal(data, config)
	if err != nil {
		return err
	}
	return nil
}
