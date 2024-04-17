package logutils

import (
	"github.com/sirupsen/logrus"
)

// Log is the logger used by the package.
var Log = logrus.New()

// Fields is the type of logrus.Fields.
type Fields = logrus.Fields

//nolint:gochecknoinits // This is the only place where we should set the log level.
func init() {
	Log.SetLevel(logrus.WarnLevel)
	Log.SetFormatter(&logrus.TextFormatter{
		TimestampFormat:           "2006-01-02 15:04:05",
		ForceColors:               true,
		EnvironmentOverrideColors: true,
		FullTimestamp:             true,
		// DisableLevelTruncation:    true,
	})
	Log.SetReportCaller(true)
}
