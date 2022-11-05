package codec

import (
	"errors"
	"log"
	"strconv"
	"strings"

	"github.com/rondoval/GoHeishaMon/topics"
)

var panasonicSetCommand = [PANASONIC_QUERY_SIZE]byte{0xf1, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

func OnAquareaCommand(mqttTopic, payload string, allTopics *topics.TopicData) *topics.TopicEntry {
	topicPieces := strings.Split(mqttTopic, "/")
	function := topicPieces[len(topicPieces)-2]
	log.Printf("Command received - set %s on %s to %s\n", function, allTopics.Kind(), payload)

	var command []byte
	switch allTopics.Kind() {
	case topics.Main:
		command = make([]byte, PANASONIC_QUERY_SIZE)
		copy(command, panasonicSetCommand[:])
	case topics.Optional:
		command = optionalPCBQuery[:] //this is not copied as we need all fields filled in
	}

	if sensor, ok := updateCommandMessage(function, payload, allTopics, command); ok {
		commandChannel <- command[:]
		return sensor
	}
	return nil
}

func verboseToNumber(value string, sensor *topics.TopicEntry) (int, error) {
	if number, err := strconv.ParseInt(value, 10, 16); err == nil {
		return int(number), nil
	}

	for valueKey, valueName := range sensor.Values {
		if value == valueName {
			return valueKey, nil
		}
	}
	return 0, errors.New("Can't convert literal to number")
}

func updateCommandMessage(function, msg string, topics *topics.TopicData, command []byte) (sensor *topics.TopicEntry, sensorOK bool) {
	sensor, sensorOK = topics.Lookup(function)
	if !sensorOK {
		log.Println("Unknown topic: " + function)
		return
	}
	if sensor.EncodeFunction == "" {
		log.Println("No encode function specified: " + function)
		return
	}

	// Update topic data as well for quicker turnaround.
	// Also needed for Optional PCB - the pump does not confirm messages.
	sensor.CurrentValue = msg

	if handler, ok := encodeInt[sensor.EncodeFunction]; ok {
		v, err := verboseToNumber(msg, sensor)
		if err != nil {
			log.Println(err)
			return sensor, false
		}
		data := handler(v, command[sensor.DecodeOffset])
		log.Printf("Setting offset %d to %d", sensor.DecodeOffset, data)
		command[sensor.DecodeOffset] = data
		return sensor, true
	} else if handler, ok := encodeFloat[sensor.EncodeFunction]; ok {
		v, err := strconv.ParseFloat(msg, 64)
		if err != nil {
			log.Println(err)
			return sensor, false
		}
		data := handler(v)
		log.Printf("Setting offset %d to %d", sensor.DecodeOffset, data)
		command[sensor.DecodeOffset] = data
		return sensor, true
	} else {
		log.Println("No encoder implemented for " + sensor.EncodeFunction)
		return sensor, false
	}
}
