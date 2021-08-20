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

type mqttSelect struct {
	Name              string   `json:"name,omitempty"`
	CommandTopic      string   `json:"command_topic,omitempty"`
	StateTopic        string   `json:"state_topic,omitempty"`
	AvailabilityTopic string   `json:"availability_topic,omitempty"`
	Options           []string `json:"options,omitempty"`
	UniqueID          string   `json:"unique_id,omitempty"`
	Device            struct {
		Manufacturer string `json:"manufacturer,omitempty"`
		Model        string `json:"model,omitempty"`
		Name         string `json:"name,omitempty"`
		Identifiers  string `json:"identifiers,omitempty"`
	} `json:"device"`
}

type mqttNumber struct {
	Name              string `json:"name,omitempty"`
	CommandTopic      string `json:"command_topic,omitempty"`
	StateTopic        string `json:"state_topic,omitempty"`
	AvailabilityTopic string `json:"availability_topic,omitempty"`
	Min               int    `json:"min,omitempty"`
	Max               int    `json:"max,omitempty"`
	Step              int    `json:"step,omitempty"`
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

func encodeSensor(sensorName, deviceID, unit string) (topic string, data []byte, err error) {
	var s mqttSensor
	s.Name = strings.ReplaceAll(sensorName, "_", " ")
	s.StateTopic = getStatusTopic(sensorName)
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

func encodeBinarySensor(sensorName, deviceID, payloadOn, payloadOff string) (topic string, data []byte, err error) {
	var s mqttBinarySensor
	s.Name = strings.ReplaceAll(sensorName, "_", " ")
	s.StateTopic = getStatusTopic(sensorName)
	s.AvailabilityTopic = config.mqttWillTopic
	s.PayloadOff = payloadOff
	s.PayloadOn = payloadOn
	s.UniqueID = deviceID + "_" + sensorName
	s.Device.Manufacturer = "Panasonic"
	s.Device.Model = "Aquarea"
	s.Device.Identifiers = deviceID
	s.Device.Name = "Aquarea " + deviceID

	topic = fmt.Sprintf("homeassistant/binary_sensor/%s/%s/config", deviceID, sensorName)
	data, err = json.Marshal(s)

	return topic, data, err
}

func encodeSwitch(sensorName, deviceID string, values []string) (topic string, data []byte, err error) {
	var b mqttSwitch
	b.Name = strings.ReplaceAll(sensorName, "_", " ")
	b.StateTopic = getStatusTopic(sensorName)
	b.CommandTopic = b.StateTopic + "/set"
	b.AvailabilityTopic = config.mqttWillTopic
	b.PayloadOn = values[1]
	b.PayloadOff = values[0]
	b.UniqueID = deviceID + "_" + sensorName
	b.Device.Manufacturer = "Panasonic"
	b.Device.Model = "Aquarea"
	b.Device.Identifiers = deviceID
	b.Device.Name = "Aquarea " + deviceID

	topic = fmt.Sprintf("homeassistant/switch/%s/%s/config", deviceID, sensorName)
	data, err = json.Marshal(b)

	return topic, data, err
}

func encodeSelect(sensorName, deviceID string, values []string) (topic string, data []byte, err error) {
	var b mqttSelect
	b.Name = strings.ReplaceAll(sensorName, "_", " ")
	b.StateTopic = getStatusTopic(sensorName)
	b.CommandTopic = b.StateTopic + "/set"
	b.AvailabilityTopic = config.mqttWillTopic
	b.Options = values
	b.UniqueID = deviceID + "_" + sensorName
	b.Device.Manufacturer = "Panasonic"
	b.Device.Model = "Aquarea"
	b.Device.Identifiers = deviceID
	b.Device.Name = "Aquarea " + deviceID

	topic = fmt.Sprintf("homeassistant/select/%s/%s/config", deviceID, sensorName)
	data, err = json.Marshal(b)

	return topic, data, err
}

func encodeNumber(sensorName, deviceID string, min, max, step int) (topic string, data []byte, err error) {
	var s mqttNumber
	s.Name = strings.ReplaceAll(sensorName, "_", " ")
	s.StateTopic = getStatusTopic(sensorName)
	s.AvailabilityTopic = config.mqttWillTopic
	s.UniqueID = deviceID + "_" + sensorName
	s.Device.Manufacturer = "Panasonic"
	s.Device.Model = "Aquarea"
	s.Device.Identifiers = deviceID
	s.Device.Name = "Aquarea " + deviceID

	topic = fmt.Sprintf("homeassistant/number/%s/%s/config", deviceID, sensorName)
	data, err = json.Marshal(s)

	return topic, data, err
}

func publishDiscoveryTopics(mclient mqtt.Client) {
	log.Print("Publishing Home Assistant discovery topics...")
	for _, value := range allTopics {
		var topic string
		var data []byte
		var err error

		if value.EncodeFunction != "" {
			// Read-Write value
			if len(value.Values) > 2 || !(value.Values[0] == "Off" || value.Values[0] == "Disabled" || value.Values[0] == "Inactive") {
				topic, data, err = encodeSelect(value.SensorName, config.DeviceName, value.Values)
			} else if len(value.Values) == 2 {
				topic, data, err = encodeSwitch(value.SensorName, config.DeviceName, value.Values)
			} else if len(value.Values) == 0 {
				topic, data, err = encodeNumber(value.SensorName, config.DeviceName, value.Min, value.Max, value.Step)
			} else {
				log.Println("Warning: Don't know how to encode " + value.SensorName)
			}
		} else {
			// Read only value
			if len(value.Values) == 2 && (value.Values[0] == "Off" || value.Values[0] == "Disabled" || value.Values[0] == "Inactive") {
				topic, data, err = encodeBinarySensor(value.SensorName, config.DeviceName, value.Values[1], value.Values[0])
			} else {
				topic, data, err = encodeSensor(value.SensorName, config.DeviceName, value.DisplayUnit)
			}
		}
		if err != nil {
			log.Println(err)
			continue
		}

		mqttPublish(mclient, topic, data, 0)
	}
	log.Println(" done.")
}
