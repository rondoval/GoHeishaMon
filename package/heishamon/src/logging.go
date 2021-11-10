package main

import (
	"io"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	gsyslog "github.com/hashicorp/go-syslog"
)

type mLogger struct {
	mclient mqtt.Client
}

var logger mLogger

func (m mLogger) Write(p []byte) (n int, err error) {
	mqttPublish(m.mclient, config.mqttLogTopic, p, 0)
	return len(p), nil
}

func logHex(command []byte) {
	if config.LogHexDump {
		log.Printf("%X\n", command)
	}
}

func redirectLogSyslog() {
	syslog, err := gsyslog.NewLogger(gsyslog.LOG_INFO, "user", "heishamon")
	if err == nil {
		log.SetOutput(syslog)
	}
}

func redirectLogMQTT(mclient mqtt.Client) {
	logger.mclient = mclient

	if config.LogMqtt == true {
		log.Println("Enabling logging to MQTT")
		log.SetOutput(io.MultiWriter(logger, log.Writer()))
	}
}
