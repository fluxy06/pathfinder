package logger

import (
	"os"

	log "github.com/sirupsen/logrus"
)

func InitLogger() *log.Logger {
	logger := log.New()
	logger.Out = os.Stdout
	logger.SetFormatter(&log.TextFormatter{
		FullTimestamp: true,
	})
	return logger
}
