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

type mqttCommon struct {
	// These are common
	Name              string     `json:"name,omitempty"`
	StateTopic        string     `json:"state_topic,omitempty"`
	AvailabilityTopic string     `json:"availability_topic,omitempty"`
	Device            mqttDevice `json:"device"`
	UniqueID          string     `json:"unique_id,omitempty"`
	EntityCategory    string     `json:"entity_category,omitempty"`
	Icon              string     `json:"icon,omitempty"`
	Qos               int        `json:"qos,omitempty"`

	// These are specific to entity types
	CommandTopic      string   `json:"command_topic,omitempty"`
	DeviceClass       string   `json:"device_class,omitempty"`
	PayloadOn         string   `json:"payload_on,omitempty"`
	PayloadOff        string   `json:"payload_off,omitempty"`
	Options           []string `json:"options,omitempty"`
	UnitOfMeasurement string   `json:"unit_of_measurement,omitempty"`
	Min               int      `json:"min,omitempty"`
	Max               int      `json:"max,omitempty"`
	Step              int      `json:"step,omitempty"`
	StateClass        string   `json:"state_class,omitempty"`
	Mode              string   `json:"mode,omitempty"` // Number
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
	case "Hz":
		return "frequency"
	case "h":
		return "duration"
	case "min":
		return "duration"
	}
	return ""
}

func encodeCommon(s *mqttCommon, info topicData, deviceID string) {
	s.Name = strings.ReplaceAll(info.SensorName, "_", " ")
	s.StateTopic = getStatusTopic(info.SensorName)
	s.AvailabilityTopic = config.mqttWillTopic
	s.Device = mqttDevice{"Panasonic", "Aquarea", "Aquarea " + deviceID, deviceID}
	s.UniqueID = deviceID + "_" + info.SensorName
	s.EntityCategory = info.Category
}

func encodeSensor(info topicData, deviceID string) (topic string, data []byte, err error) {
	var s mqttCommon
	encodeCommon(&s, info, deviceID)
	s.UnitOfMeasurement = info.DisplayUnit
	s.DeviceClass = getDeviceClass(info.DisplayUnit)

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
	var s mqttCommon
	encodeCommon(&s, info, deviceID)
	s.PayloadOff = info.Values[0]
	s.PayloadOn = info.Values[1]

	topic = fmt.Sprintf("homeassistant/binary_sensor/%s/%s/config", deviceID, info.SensorName)
	data, err = json.Marshal(s)

	return topic, data, err
}

func encodeSwitch(info topicData, deviceID string) (topic string, data []byte, err error) {
	var b mqttCommon
	encodeCommon(&b, info, deviceID)
	b.CommandTopic = b.StateTopic + "/set"
	b.PayloadOn = info.Values[1]
	b.PayloadOff = info.Values[0]

	topic = fmt.Sprintf("homeassistant/switch/%s/%s/config", deviceID, info.SensorName)
	data, err = json.Marshal(b)

	return topic, data, err
}

func encodeSelect(info topicData, deviceID string) (topic string, data []byte, err error) {
	var b mqttCommon
	encodeCommon(&b, info, deviceID)
	b.CommandTopic = b.StateTopic + "/set"
	b.Options = info.Values

	topic = fmt.Sprintf("homeassistant/select/%s/%s/config", deviceID, info.SensorName)
	data, err = json.Marshal(b)

	return topic, data, err
}

func encodeNumber(info topicData, deviceID string) (topic string, data []byte, err error) {
	var s mqttCommon
	encodeCommon(&s, info, deviceID)
	s.DeviceClass = getDeviceClass(info.DisplayUnit)
	s.CommandTopic = s.StateTopic + "/set"
	s.UnitOfMeasurement = info.DisplayUnit
	s.Min = info.Min
	s.Max = info.Max
	s.Step = info.Step
	s.Mode = "box"

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
