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
	sensorName := topicPieces[len(topicPieces)-2]
	log.Printf("Command received - set %s on %s to %s\n", sensorName, allTopics.Kind(), payload)

	switch allTopics.Kind() {
	case topics.Main:
		command := make([]byte, PANASONIC_QUERY_SIZE)
		copy(command, panasonicSetCommand[:])
		if sensor, ok := updateCommandMessage(sensorName, payload, allTopics, command); ok {
			commandChannel <- command[:]
			return sensor
		}

	case topics.Optional:
		optionalPCBMutex.Lock()
		defer optionalPCBMutex.Unlock()
		//this is not copied as we need all fields filled in
		if sensor, ok := updateCommandMessage(sensorName, payload, allTopics, optionalPCBQuery[:]); ok {
			command := make([]byte, len(optionalPCBQuery))
			copy(command, optionalPCBQuery[:])
			commandChannel <- command[:]
			return sensor
		}
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

	// Update topic data as well for quicker turnaround.
	// Also needed for Optional PCB - the pump does not confirm messages.
	sensor.UpdateValue(msg)
	return sensor, encode(sensor, command)
}
