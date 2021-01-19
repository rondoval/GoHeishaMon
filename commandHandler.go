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

func onCommand(mclient mqtt.Client, msg mqtt.Message) {
	topicPieces := strings.Split(msg.Topic(), "/")
	function := topicPieces[len(topicPieces)-1]
	value := string(msg.Payload())
	log.Printf("Command received - set %s to %s\n", function, value)

	if function == "OSCommand" {
		handleOSCommand(mclient, msg)
		return
	}

	command, err := prepMainCommand(function, value)
	if err == nil {
		commandsChannel <- command[:]
	} else if config.OptionalPCB == true {
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
	} else {
		log.Printf("Unknown command (%s) or value conversion error (%s)", function, value)
	}
}

func verboseToNumber(command, value string) (int64, error) {
	for _, topic := range allTopics {
		if topic.Command == command {
			for valueKey, valueName := range topic.Values {
				if value == valueName {
					return int64(valueKey), nil
				}
			}
		}
	}
	return 0, errors.New("Can't convert literal to number")
}

func prepMainCommand(name, msg string) ([setCmdLen]byte, error) {
	if name == "SetCurves" {
		return setCurves(msg)
	}

	command := panasonicSetCommand
	v, err := strconv.ParseInt(msg, 10, 8)
	if err != nil {
		v, err = verboseToNumber(name, msg)
		if err != nil {
			return command, err
		}
	}

	if handler, ok := mainCommandMap[name]; ok {
		data, index := handler(byte(v))
		command[index] = data
	}

	return command, nil
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
