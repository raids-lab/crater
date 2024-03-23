package logutils

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// Log is the logger used by the package.
var Log = logrus.New()

// Fields is the type of logrus.Fields.
type Fields = logrus.Fields

//nolint:gochecknoinits // This is the only place where we should set the log level.
func init() {
	if gin.Mode() == gin.DebugMode {
		Log.SetLevel(logrus.DebugLevel)
	} else {
		Log.SetLevel(logrus.InfoLevel)
	}
	Log.SetFormatter(&logrus.TextFormatter{
		TimestampFormat:           "2006-01-02 15:04:05",
		ForceColors:               true,
		EnvironmentOverrideColors: true,
		FullTimestamp:             true,
		// DisableLevelTruncation:    true,
	})
	Log.SetReportCaller(true)
}
