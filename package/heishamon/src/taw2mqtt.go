// Package main is the acutal Heishamon program.
package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rondoval/GoHeishaMon/codec"
	"github.com/rondoval/GoHeishaMon/logger"
	"github.com/rondoval/GoHeishaMon/mqtt"
	"github.com/rondoval/GoHeishaMon/serial"
	"github.com/rondoval/GoHeishaMon/topics"
)

func main() {
	var configPath = flag.String("path", "/etc/heishamon", "Path to Heishamon configuration files")
	flag.Parse()

	logger.Configure()
	log.Println("GoHeishaMon loading...")

	var config configStruct
	config.readConfig(*configPath)
	logger.SetLevel(config.LogHexDump, config.LogDebug)

	commandTopics := topics.LoadTopics(config.topicsFile, config.getDeviceName(topics.Main), topics.Main)
	optionalPCBTopics := topics.LoadTopics(config.topicsOptionalPCBFile, config.getDeviceName(topics.Optional), topics.Optional)

	mclient := mqtt.MakeMQTTConn(mqtt.Options{
		Server:         config.MqttServer,
		Port:           config.MqttPort,
		Username:       config.MqttLogin,
		Password:       config.MqttPass,
		BaseTopic:      config.MqttTopicBase,
		KeepAlive:      time.Second * time.Duration(config.MqttKeepalive),
		ListenOnly:     config.ListenOnly,
		OptionalPCB:    config.OptionalPCB,
		CommandTopics:  commandTopics,
		OptionalTopics: optionalPCBTopics,
	})
	if config.LogMqtt {
		logger.RedirectLogMQTT(&mclient)
	}

	if config.HAAutoDiscover {
		mclient.PublishDiscoveryTopics(commandTopics)
		if config.OptionalPCB {
			mclient.PublishDiscoveryTopics(optionalPCBTopics)
		}
	}

	if config.OptionalPCB {
		log.Println("Restoring Optional PCB values")
		changed := optionalPCBTopics.Unmarshal(config.optionalPCBFile)
		for _, c := range changed {
			mclient.PublishValue(c)
		}
		if config.OptionalSaveInterval > 0 {
			go func() {
				log.Println("PCB save thread starting")
				for range time.Tick(time.Minute * time.Duration(config.OptionalSaveInterval)) {
					optionalPCBTopics.Marshal(config.optionalPCBFile)
				}
			}()
		}
	}

	var serialPort serial.Comms
	serialPort.Open(config.SerialPort, time.Millisecond*time.Duration(config.SerialTimeout))
	defer serialPort.Close()

	receivedChannel := make(chan bool)
	acknowledgeChannel := make(chan []byte)
	commandChannel := codec.Start(codec.Options{
		MQTT:          mclient,
		QueryInterval: time.Duration(config.QueryInterval),
		AckChannel:    acknowledgeChannel,

		OptionalPCB:           config.OptionalPCB,
		OptionalQueryInterval: time.Duration(config.OptionalQueryInterval),
		OptionalTopics:        optionalPCBTopics,
	})

	go func() {
		log.Println("Receiver thread starting")
		for {
			data := serialPort.Read(config.LogHexDump)
			if data != nil {
				select {
				case receivedChannel <- true:
				default:
				}
			}
			if len(data) == serial.OptionalMessageLength {
				values := codec.Decode(optionalPCBTopics, data)
				for _, v := range values {
					mclient.PublishValue(v)
				}
				acknowledgeChannel <- data
			} else if len(data) == serial.DataMessageLength {
				values := codec.Decode(commandTopics, data)
				for _, v := range values {
					mclient.PublishValue(v)
				}
			} else if data != nil {
				logger.LogDebug("Unknown message length: %d", len(data))
			}
		}
	}()

	shutdownChannel := make(chan os.Signal, 1)
	signal.Notify(shutdownChannel, syscall.SIGTERM)
	shallTerminate := false

	log.Print("Entering main loop")
	for {
		command := <-commandChannel
		serialPort.SendCommand(command)
		if shallTerminate && (!config.OptionalPCB || len(command) == codec.OptionalDatagramSize) {
			return
		}

		select {
		case <-shutdownChannel:
			log.Print("SIGTERM received")
			shallTerminate = true
		case <-receivedChannel:
			// ok, did receive something, can send next request
		case <-time.After(15 * time.Second):
			log.Println("Response not received, recovering")
		}

		var queueLen = len(commandChannel)
		if queueLen > 10 {
			log.Print("Command queue length: ", queueLen)
		}
	}
}
