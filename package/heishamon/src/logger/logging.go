package logger

import (
	"errors"
	"io"
	"log"

	//	gsyslog "github.com/hashicorp/go-syslog"
	"github.com/rondoval/GoHeishaMon/mqtt"
)

type mLogger struct {
	mclient   *mqtt.MQTT
	mqttTopic string
	logDebug  bool
	logHex    bool
}

var logger mLogger

// Writer used to send log messages via MQTT
func (m mLogger) Write(p []byte) (n int, err error) {
	if m.mclient == nil {
		return 0, errors.New("No MQTT client")
	}
	m.mclient.Publish(m.mqttTopic, p, 0)
	return len(p), nil
}

// SetLevel sets the logging level, i.e. enables/disables debug logging and datagram logging.
func SetLevel(loghex, logdebug bool) {
	logger.logDebug = logdebug
	logger.logHex = loghex
}

// LogHex logs heat pump datagrams. Controlled by logHex config option.
func LogHex(comment string, command []byte) {
	if logger.logHex {
		log.Printf("%s: %X\n", comment, command)
	}
}

// LogDebug logs debugging info. Controlled by logDebug config option.
func LogDebug(format string, v ...any) {
	if logger.logDebug {
		log.Printf(format, v...)
	}
}

// Configure sets up logging mechanism and enables syslog logging.
func Configure() {
	log.SetFlags(log.Lshortfile)
	// syslog, err := gsyslog.NewLogger(gsyslog.LOG_INFO, "user", "heishamon")
	// if err == nil {
	// 	log.SetOutput(syslog)
	// }
}

// RedirectLogMQTT enables MQTT logging. Does not disable syslog logging.
func RedirectLogMQTT(mclient *mqtt.MQTT) {
	logger.mclient = mclient
	logger.mqttTopic = mclient.LogTopic()

	log.Println("Enabling logging to MQTT")
	log.SetOutput(io.MultiWriter(logger, log.Writer()))
}
