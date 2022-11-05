package logger

import (
	"errors"
	"io"
	"log"

	gsyslog "github.com/hashicorp/go-syslog"
	"github.com/rondoval/GoHeishaMon/mqtt"
)

type mLogger struct {
	mclient   *mqtt.MQTT
	mqttTopic string
	logDebug  bool
	logHex    bool
}

var logger mLogger

func (m mLogger) Write(p []byte) (n int, err error) {
	if m.mclient == nil {
		return 0, errors.New("No MQTT client")
	}
	m.mclient.Publish(m.mqttTopic, p, 0)
	return len(p), nil
}

func SetLevel(loghex, logdebug bool) {
	logger.logDebug = logdebug
	logger.logHex = loghex
}

func LogHex(command []byte) {
	if logger.logHex {
		log.Printf("%X\n", command)
	}
}

func LogDebug(format string, v ...any) {
	if logger.logDebug {
		log.Printf(format, v...)
	}
}

func Configure() {
	log.SetFlags(log.Lshortfile)
	syslog, err := gsyslog.NewLogger(gsyslog.LOG_INFO, "user", "heishamon")
	if err == nil {
		log.SetOutput(syslog)
	}
}

func RedirectLogMQTT(mclient *mqtt.MQTT) {
	logger.mclient = mclient
	logger.mqttTopic = mclient.LogTopic()

	log.Println("Enabling logging to MQTT")
	log.SetOutput(io.MultiWriter(logger, log.Writer()))
}
