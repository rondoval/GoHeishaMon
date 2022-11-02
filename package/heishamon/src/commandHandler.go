package main

import (
	"errors"
	"log"
	"strconv"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const setCmdLen = 110

var panasonicSetCommand = [setCmdLen]byte{0xf1, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

func onAquareaCommand(mclient mqtt.Client, msg mqtt.Message) {
	topicPieces := strings.Split(msg.Topic(), "/")
	device := topicPieces[len(topicPieces)-3] // main, optional
	function := topicPieces[len(topicPieces)-2]
	value := string(msg.Payload())
	log.Printf("Command received - set %s on %s to %s\n", function, device, value)

	var topics topicData
	var command []byte
	if device == main_topic {
		topics = commandTopics
		command = make([]byte, setCmdLen)
		copy(command, panasonicSetCommand[:])
	} else if device == optional_topic {
		topics = optionalPCBTopics
		command = optionalPCBQuery //this is not copied as we need all fields filled in
	}

	err := updateCommandMessage(function, value, topics, command)
	if err == nil {
		commandsChannel <- command[:]
	} else {
		log.Print(err)
	}
}

func verboseToNumber(function, value string, topics topicData) (int64, error) {
	if sensor, ok := topics.lookup(function); ok {
		for valueKey, valueName := range sensor.Values {
			if value == valueName {
				return int64(valueKey), nil
			}
		}
	}
	return 0, errors.New("Can't convert literal to number")
}

func updateCommandMessage(function, msg string, topics topicData, command []byte) error {
	v, err := strconv.ParseInt(msg, 10, 16)
	if err != nil {
		v, err = verboseToNumber(function, msg, topics)
		if err != nil {
			return err
		}
	}

	if sensor, ok := topics.lookup(function); ok {
		if sensor.EncodeFunction != "" {
			if handler, ok := encodeInt[sensor.EncodeFunction]; ok {
				data := handler(int(v), command[sensor.DecodeOffset])
				log.Printf("Setting offset %d to %d", sensor.DecodeOffset, data)
				command[sensor.DecodeOffset] = data
				return nil
			}
			return errors.New("Unknown command " + sensor.EncodeFunction)
		}
		return errors.New("No encode function defined for " + function)
	}
	return errors.New("Unknown topic " + function)
}
