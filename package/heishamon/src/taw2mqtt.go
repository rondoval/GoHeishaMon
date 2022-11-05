package main

import (
	"flag"
	"log"
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

	var serialPort serial.SerialComms
	serialPort.Open(config.SerialPort, time.Millisecond*time.Duration(config.SerialTimeout))
	defer serialPort.Close()
	commandChannel := codec.GetChannel()
	commandTopics := topics.LoadTopics(config.topicsFile, config.getDeviceName(topics.Main), topics.Main)
	optionalPCBTopics := topics.LoadTopics(config.topicsOptionalPCBFile, config.getDeviceName(topics.Optional), topics.Optional)

	if config.OptionalPCB {
		codec.LoadOptionalPCB(config.optionalPCBFile)
	}

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
	if config.HAAutoDiscover == true {
		mclient.PublishDiscoveryTopics(commandTopics)
		if config.OptionalPCB == true {
			mclient.PublishDiscoveryTopics(optionalPCBTopics)
		}
	}

	if config.OptionalPCB {
		optionPCBSaveTicker := time.NewTicker(time.Minute * time.Duration(config.OptionalSaveInterval))
		go func() {
			log.Println("PCB save thread starting")
			for {
				<-optionPCBSaveTicker.C
				codec.SaveOptionalPCB(config.optionalPCBFile)
			}
		}()
	}

	go func() {
		log.Println("Receiver thread starting")
		for {
			data := serialPort.Read(config.LogHexDump)
			if len(data) == serial.OPTIONAL_MSG_LENGTH {
				values := codec.DecodeHeatpumpData(optionalPCBTopics, data)
				for _, v := range values {
					mclient.PublishValue(v)
				}
				codec.Acknowledge(data)
			} else if len(data) == serial.DATA_MSG_LENGTH {
				values := codec.DecodeHeatpumpData(commandTopics, data)
				for _, v := range values {
					mclient.PublishValue(v)
				}
			} else if data != nil {
				logger.LogDebug("Unkown message length: %d", len(data))
			}
		}
	}()

	queryTicker := time.NewTicker(time.Second * time.Duration(config.QueryInterval))
	optionQueryTicker := time.NewTicker(time.Second * time.Duration(config.OptionalQueryInterval))

	log.Print("Entering main loop")
	for {
		// TODO combine requests into single command if many coming from MQTT?
		var queueLen = len(commandChannel)
		if queueLen > 10 {
			log.Print("Command queue length: ", queueLen)
		}

		select {
		case value := <-commandChannel:
			switch len(value) {
			case codec.PANASONIC_QUERY_SIZE:
				queryTicker.Reset(time.Second * time.Duration(config.QueryInterval))

			case codec.OPTIONAL_QUERY_SIZE:
				optionQueryTicker.Reset(time.Second * time.Duration(config.OptionalQueryInterval))
			}
			serialPort.SendCommand(value)
			//TODO this should likely be sync'd with following read

		case <-optionQueryTicker.C:
			if config.OptionalPCB == true && config.ListenOnly == false {
				codec.SendOptionalPCBQuery()
			}

		case <-queryTicker.C:
			codec.SendPanasonicQuery()
		}
	}
}
