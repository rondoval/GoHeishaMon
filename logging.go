package main

import (
	"io"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
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

func redirectLog(mclient mqtt.Client) {
	logger.mclient = mclient

	if config.LogMqtt == true {
		log.Println("Enabling logging to MQTT")
		log.SetOutput(io.MultiWriter(log.Writer(), logger))
	}
}
