package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type mqttSwitch struct {
	Name              string `json:"name,omitempty"`
	CommandTopic      string `json:"command_topic,omitempty"`
	StateTopic        string `json:"state_topic,omitempty"`
	AvailabilityTopic string `json:"availability_topic,omitempty"`
	PayloadOn         string `json:"payload_on,omitempty"`
	PayloadOff        string `json:"payload_off,omitempty"`
	UniqueID          string `json:"unique_id,omitempty"`
	Device            struct {
		Manufacturer string `json:"manufacturer,omitempty"`
		Model        string `json:"model,omitempty"`
		Name         string `json:"name,omitempty"`
		Identifiers  string `json:"identifiers,omitempty"`
	} `json:"device"`
}

type mqttSensor struct {
	Name              string `json:"name,omitempty"`
	StateTopic        string `json:"state_topic"`
	AvailabilityTopic string `json:"availability_topic,omitempty"`
	UnitOfMeasurement string `json:"unit_of_measurement,omitempty"`
	DeviceClass       string `json:"device_class,omitempty"`
	ForceUpdate       bool   `json:"force_update,omitempty"`
	ExpireAfter       int    `json:"expire_after,omitempty"`
	UniqueID          string `json:"unique_id,omitempty"`
	Device            struct {
		Manufacturer string `json:"manufacturer,omitempty"`
		Model        string `json:"model,omitempty"`
		Name         string `json:"name,omitempty"`
		Identifiers  string `json:"identifiers,omitempty"`
	} `json:"device"`
}

type mqttBinarySensor struct {
	Name              string `json:"name,omitempty"`
	StateTopic        string `json:"state_topic"`
	AvailabilityTopic string `json:"availability_topic,omitempty"`
	PayloadOn         string `json:"payload_on,omitempty"`
	PayloadOff        string `json:"payload_off,omitempty"`
	DeviceClass       string `json:"device_class,omitempty"`
	ForceUpdate       bool   `json:"force_update,omitempty"`
	ExpireAfter       int    `json:"expire_after,omitempty"`
	UniqueID          string `json:"unique_id,omitempty"`
	Device            struct {
		Manufacturer string `json:"manufacturer,omitempty"`
		Model        string `json:"model,omitempty"`
		Name         string `json:"name,omitempty"`
		Identifiers  string `json:"identifiers,omitempty"`
	} `json:"device"`
}

func getDeviceClass(unit string) string {
	switch unit {
	case "W":
		return "power"
	case "kW":
		return "power"
	case "Wh":
		return "energy"
	case "kWh":
		return "energy"
	case "A":
		return "current"
	case "°C":
		return "temperature"
	}
	return ""
}

func encodeSensor(sensorName, deviceID, stateTopic, unit string) (topic string, data []byte, err error) {
	var s mqttSensor
	s.Name = strings.ReplaceAll(sensorName, "_", " ")
	s.StateTopic = stateTopic
	s.AvailabilityTopic = config.mqttWillTopic
	s.UnitOfMeasurement = unit
	s.DeviceClass = getDeviceClass(unit)
	s.UniqueID = deviceID + "_" + sensorName
	s.Device.Manufacturer = "Panasonic"
	s.Device.Model = "Aquarea"
	s.Device.Identifiers = deviceID
	s.Device.Name = "Aquarea " + deviceID

	topic = fmt.Sprintf("homeassistant/sensor/%s/%s/config", deviceID, sensorName)
	data, err = json.Marshal(s)

	return topic, data, err
}

func encodeBinarySensor(sensorName, deviceID, stateTopic, payloadOn, payloadOff string) (topic string, data []byte, err error) {
	var s mqttBinarySensor
	s.Name = strings.ReplaceAll(sensorName, "_", " ")
	s.StateTopic = stateTopic
	s.AvailabilityTopic = config.mqttWillTopic
	s.PayloadOff = payloadOff
	s.PayloadOn = payloadOn
	s.UniqueID = deviceID + "_" + sensorName
	s.Device.Manufacturer = "Panasonic"
	s.Device.Model = "Aquarea"
	s.Device.Identifiers = deviceID
	s.Device.Name = "Aquarea " + deviceID

	topic = fmt.Sprintf("homeassistant/sensor/%s/%s/config", deviceID, sensorName)
	data, err = json.Marshal(s)

	return topic, data, err
}

func encodeSwitch(commandName, deviceID, sensorName string, values []string) (topic string, data []byte, err error) {
	var b mqttSwitch
	b.Name = commandName
	b.CommandTopic = getCommandTopic(commandName)
	b.StateTopic = getStatusTopic(sensorName)
	b.AvailabilityTopic = config.mqttWillTopic
	b.PayloadOn = values[0]
	b.PayloadOff = values[1]
	b.UniqueID = deviceID + "_" + commandName
	b.Device.Manufacturer = "Panasonic"
	b.Device.Model = "Aquarea"
	b.Device.Identifiers = deviceID
	b.Device.Name = "Aquarea " + deviceID

	topic = fmt.Sprintf("homeassistant/switch/%s/%s/config", deviceID, commandName)
	data, err = json.Marshal(b)

	return topic, data, err
}

func publishDiscoveryTopics(mclient mqtt.Client) {
	for _, value := range allTopics {
		stateTopic := getStatusTopic(value.SensorName)
		var topic string
		var data []byte
		var err error
		if len(value.Values) != 2 || !(value.Values[0] == "Off" || value.Values[0] == "Disabled") {
			topic, data, err = encodeSensor(value.SensorName, config.DeviceName, stateTopic, value.DisplayUnit)
		} else {
			topic, data, err = encodeBinarySensor(value.SensorName, config.DeviceName, stateTopic, value.Values[1], value.Values[0])
		}
		if err != nil {
			log.Println(err)
			continue
		}

		token := mclient.Publish(topic, 0, true, data)
		if token.Wait() && token.Error() != nil {
			log.Printf("Failed to publish, %v", token.Error())
			continue
		}

		// OK, we have a sensor. Now check if there's an associated command
		if value.Command != "" {
			topic, data, err = encodeSwitch(value.Command, config.DeviceName, value.SensorName, value.Values)
			if err != nil {
				log.Println(err)
				continue
			}

			token = mclient.Publish(topic, 0, true, data)
			if token.Wait() && token.Error() != nil {
				log.Printf("Fail to publish, %v", token.Error())
			}
		}
	}
}
