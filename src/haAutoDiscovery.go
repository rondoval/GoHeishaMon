package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type mqttDevice struct {
	Manufacturer string `json:"manufacturer,omitempty"`
	Model        string `json:"model,omitempty"`
	Name         string `json:"name,omitempty"`
	Identifiers  string `json:"identifiers,omitempty"`
}

type mqttSwitch struct {
	Name              string     `json:"name,omitempty"`
	CommandTopic      string     `json:"command_topic,omitempty"`
	StateTopic        string     `json:"state_topic,omitempty"`
	AvailabilityTopic string     `json:"availability_topic,omitempty"`
	PayloadOn         string     `json:"payload_on,omitempty"`
	PayloadOff        string     `json:"payload_off,omitempty"`
	UniqueID          string     `json:"unique_id,omitempty"`
	EntityCategory    string     `json:"entity_category,omitempty"`
	Device            mqttDevice `json:"device"`
}

type mqttSelect struct {
	Name              string     `json:"name,omitempty"`
	CommandTopic      string     `json:"command_topic,omitempty"`
	StateTopic        string     `json:"state_topic,omitempty"`
	AvailabilityTopic string     `json:"availability_topic,omitempty"`
	Options           []string   `json:"options,omitempty"`
	UniqueID          string     `json:"unique_id,omitempty"`
	EntityCategory    string     `json:"entity_category,omitempty"`
	Device            mqttDevice `json:"device"`
}

type mqttNumber struct {
	Name              string     `json:"name,omitempty"`
	CommandTopic      string     `json:"command_topic,omitempty"`
	StateTopic        string     `json:"state_topic,omitempty"`
	AvailabilityTopic string     `json:"availability_topic,omitempty"`
	UnitOfMeasurement string     `json:"unit_of_measurement,omitempty"`
	Min               int        `json:"min,omitempty"`
	Max               int        `json:"max,omitempty"`
	Step              int        `json:"step,omitempty"`
	UniqueID          string     `json:"unique_id,omitempty"`
	EntityCategory    string     `json:"entity_category,omitempty"`
	Device            mqttDevice `json:"device"`
}

type mqttSensor struct {
	Name              string     `json:"name,omitempty"`
	StateTopic        string     `json:"state_topic"`
	AvailabilityTopic string     `json:"availability_topic,omitempty"`
	UnitOfMeasurement string     `json:"unit_of_measurement,omitempty"`
	DeviceClass       string     `json:"device_class,omitempty"`
	StateClass        string     `json:"state_class,omitempty"`
	ForceUpdate       bool       `json:"force_update,omitempty"`
	ExpireAfter       int        `json:"expire_after,omitempty"`
	UniqueID          string     `json:"unique_id,omitempty"`
	EntityCategory    string     `json:"entity_category,omitempty"`
	Device            mqttDevice `json:"device"`
}

type mqttBinarySensor struct {
	Name              string     `json:"name,omitempty"`
	StateTopic        string     `json:"state_topic"`
	AvailabilityTopic string     `json:"availability_topic,omitempty"`
	PayloadOn         string     `json:"payload_on,omitempty"`
	PayloadOff        string     `json:"payload_off,omitempty"`
	DeviceClass       string     `json:"device_class,omitempty"`
	ForceUpdate       bool       `json:"force_update,omitempty"`
	ExpireAfter       int        `json:"expire_after,omitempty"`
	UniqueID          string     `json:"unique_id,omitempty"`
	EntityCategory    string     `json:"entity_category,omitempty"`
	Device            mqttDevice `json:"device"`
}

func getMqttDevice(deviceID string) mqttDevice {
	return mqttDevice{"Panasonic", "Aquarea", "Aquarea " + deviceID, deviceID}
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

func encodeSensor(info topicData, deviceID string) (topic string, data []byte, err error) {
	var s mqttSensor
	s.Name = strings.ReplaceAll(info.SensorName, "_", " ")
	s.StateTopic = getStatusTopic(info.SensorName)
	s.AvailabilityTopic = config.mqttWillTopic
	s.UnitOfMeasurement = info.DisplayUnit
	s.DeviceClass = getDeviceClass(info.DisplayUnit)
	s.UniqueID = deviceID + "_" + info.SensorName
	s.EntityCategory = info.Category
	s.Device = getMqttDevice(deviceID)

	switch s.UnitOfMeasurement {
	case "h", "Counter":
		s.StateClass = "total_increasing"
	case "ErrorState", "":
		// nothing
	default:
		s.StateClass = "measurement"
	}
	topic = fmt.Sprintf("homeassistant/sensor/%s/%s/config", deviceID, info.SensorName)
	data, err = json.Marshal(s)

	return topic, data, err
}

func encodeBinarySensor(info topicData, deviceID string) (topic string, data []byte, err error) {
	var s mqttBinarySensor
	s.Name = strings.ReplaceAll(info.SensorName, "_", " ")
	s.StateTopic = getStatusTopic(info.SensorName)
	s.AvailabilityTopic = config.mqttWillTopic
	s.PayloadOff = info.Values[0]
	s.PayloadOn = info.Values[1]
	s.UniqueID = deviceID + "_" + info.SensorName
	s.EntityCategory = info.Category
	s.Device = getMqttDevice(deviceID)

	topic = fmt.Sprintf("homeassistant/binary_sensor/%s/%s/config", deviceID, info.SensorName)
	data, err = json.Marshal(s)

	return topic, data, err
}

func encodeSwitch(info topicData, deviceID string) (topic string, data []byte, err error) {
	var b mqttSwitch
	b.Name = strings.ReplaceAll(info.SensorName, "_", " ")
	b.StateTopic = getStatusTopic(info.SensorName)
	b.CommandTopic = b.StateTopic + "/set"
	b.AvailabilityTopic = config.mqttWillTopic
	b.PayloadOn = info.Values[1]
	b.PayloadOff = info.Values[0]
	b.UniqueID = deviceID + "_" + info.SensorName
	b.EntityCategory = info.Category
	b.Device = getMqttDevice(deviceID)

	topic = fmt.Sprintf("homeassistant/switch/%s/%s/config", deviceID, info.SensorName)
	data, err = json.Marshal(b)

	return topic, data, err
}

func encodeSelect(info topicData, deviceID string) (topic string, data []byte, err error) {
	var b mqttSelect
	b.Name = strings.ReplaceAll(info.SensorName, "_", " ")
	b.StateTopic = getStatusTopic(info.SensorName)
	b.CommandTopic = b.StateTopic + "/set"
	b.AvailabilityTopic = config.mqttWillTopic
	b.Options = info.Values
	b.UniqueID = deviceID + "_" + info.SensorName
	b.EntityCategory = info.Category
	b.Device = getMqttDevice(deviceID)

	topic = fmt.Sprintf("homeassistant/select/%s/%s/config", deviceID, info.SensorName)
	data, err = json.Marshal(b)

	return topic, data, err
}

func encodeNumber(info topicData, deviceID string) (topic string, data []byte, err error) {
	var s mqttNumber
	s.Name = strings.ReplaceAll(info.SensorName, "_", " ")
	s.StateTopic = getStatusTopic(info.SensorName)
	s.CommandTopic = s.StateTopic + "/set"
	s.AvailabilityTopic = config.mqttWillTopic
	s.UnitOfMeasurement = info.DisplayUnit
	s.Min = info.Min
	s.Max = info.Max
	s.Step = info.Step
	s.UniqueID = deviceID + "_" + info.SensorName
	s.EntityCategory = info.Category
	s.Device = getMqttDevice(deviceID)

	topic = fmt.Sprintf("homeassistant/number/%s/%s/config", deviceID, info.SensorName)
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
			if len(value.Values) == 0 {
				topic, data, err = encodeNumber(value, config.DeviceName)
			} else if len(value.Values) > 2 || !(value.Values[0] == "Off" || value.Values[0] == "Disabled" || value.Values[0] == "Inactive") {
				topic, data, err = encodeSelect(value, config.DeviceName)
			} else if len(value.Values) == 2 {
				topic, data, err = encodeSwitch(value, config.DeviceName)
			} else {
				log.Println("Warning: Don't know how to encode " + value.SensorName)
			}
		} else {
			// Read only value
			if len(value.Values) == 2 && (value.Values[0] == "Off" || value.Values[0] == "Disabled" || value.Values[0] == "Inactive") {
				topic, data, err = encodeBinarySensor(value, config.DeviceName)
			} else {
				topic, data, err = encodeSensor(value, config.DeviceName)
			}
		}
		if err != nil {
			log.Print(err)
			continue
		}

		mqttPublish(mclient, topic, data, 0)
	}
	log.Println("Publishing Home Assistant discovery topics done.")
}
