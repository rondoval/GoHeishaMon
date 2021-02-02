package main

import (
	"errors"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func onGenericCommand(mclient mqtt.Client, msg mqtt.Message) {
	topicPieces := strings.Split(msg.Topic(), "/")
	function := topicPieces[len(topicPieces)-1]
	value := string(msg.Payload())
	log.Printf("Command received - set %s to %s\n", function, value)

	if function == "OSCommand" {
		handleOSCommand(mclient, msg)
		return
	}
	log.Printf("Unknown command %s", function)
}

func onAquareaCommand(mclient mqtt.Client, msg mqtt.Message) {
	topicPieces := strings.Split(msg.Topic(), "/")
	function := topicPieces[len(topicPieces)-2]
	value := string(msg.Payload())
	log.Printf("Command received - set %s to %s\n", function, value)

	command, err := prepMainCommand(function, value)
	if err == nil {
		commandsChannel <- command[:]
	} else if config.OptionalPCB == true {
		handlePCBCommand(function, value)
	} else {
		log.Println(err)
	}
}

func verboseToNumber(function, value string) (int64, error) {
	if sensor, ok := topicNameLookup[function]; ok {
		for valueKey, valueName := range sensor.Values {
			if value == valueName {
				return int64(valueKey), nil
			}
		}
	}
	return 0, errors.New("Can't convert literal to number")
}

func prepMainCommand(function, msg string) ([setCmdLen]byte, error) {
	command := panasonicSetCommand
	v, err := strconv.ParseInt(msg, 10, 16)
	if err != nil {
		v, err = verboseToNumber(function, msg)
		if err != nil {
			return command, err
		}
	}

	if sensor, ok := topicNameLookup[function]; ok {
		if sensor.EncodeFunction != "" {
			if handler, ok := encodeInt[sensor.EncodeFunction]; ok {
				data := handler(int(v))
				log.Printf("Setting offset %d to %d", sensor.DecodeOffset, data)
				command[sensor.DecodeOffset] = data
				return command, nil
			}
			return command, errors.New("Unknown command encodeFunction")
		}
		return command, errors.New("No encode function defined for this topic")
	}
	return command, errors.New("Unknown topic")
}

func handlePCBCommand(function, value string) {
	if handler, ok := optionCommandMapFloat[function]; ok {
		temp, err := strconv.ParseFloat(value, 64)
		if err != nil {
			log.Printf("%s: %s value conversion error", function, value)
			return
		}
		handler(temp)
	} else if handler, ok := optionCommandMapByte[function]; ok {
		v, err := strconv.ParseInt(value, 10, 8)
		if err != nil {
			log.Printf("%s: %s value conversion error", function, value)
			return
		}
		handler(byte(v))
	} else {
		log.Printf("Unknown command (%s) or value conversion error (%s)", function, value)
	}
}

func handleOSCommand(mclient mqtt.Client, msg mqtt.Message) {
	if config.EnableOSCommand == false {
		return
	}
	var cmd *exec.Cmd
	var out2 string
	s := strings.Split(string(msg.Payload()), " ")
	if len(s) < 2 {
		cmd = exec.Command(s[0])
	} else {
		cmd = exec.Command(s[0], s[1:]...)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		// TODO: handle error more gracefully
		out2 = fmt.Sprintf("%s", err)
	}
	comout := fmt.Sprintf("%s - %s", out, out2)
	TOP := fmt.Sprintf("%s/out", getCommandTopic(("OSCommand")))
	mqttPublish(mclient, TOP, comout, 0)
}
