package mqtt

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/rondoval/GoHeishaMon/topics"
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
	Mode              string   `json:"mode,omitempty"`

	entityType string
	sensorName string
	deviceID   string
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

func (s *mqttCommon) encodeCommon(info *topics.TopicEntry, statusTopic, willTopic, deviceName string) {
	s.Name = strings.ReplaceAll(info.SensorName, "_", " ")
	s.StateTopic = statusTopic
	s.AvailabilityTopic = willTopic

	s.sensorName = info.SensorName
	s.deviceID = strings.ReplaceAll(deviceName, " ", "_")

	s.Device = mqttDevice{"Panasonic", "Aquarea", "Aquarea " + deviceName, s.deviceID}
	s.UniqueID = s.deviceID + "_" + info.SensorName
	s.EntityCategory = info.Category
}

func (s *mqttCommon) encodeSensor(info *topics.TopicEntry) {
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

	s.entityType = "sensor"
}

func (s *mqttCommon) encodeBinarySensor(info *topics.TopicEntry) {
	s.PayloadOff = info.Values[0]
	s.PayloadOn = info.Values[1]

	s.entityType = "binary_sensor"
}

func (s *mqttCommon) encodeSwitch(info *topics.TopicEntry) {
	s.CommandTopic = s.StateTopic + "/set"
	s.PayloadOn = info.Values[1]
	s.PayloadOff = info.Values[0]

	s.entityType = "switch"
}

func (s *mqttCommon) encodeSelect(info *topics.TopicEntry) {
	s.CommandTopic = s.StateTopic + "/set"
	s.Options = info.Values

	s.entityType = "select"
}

func (s *mqttCommon) encodeNumber(info *topics.TopicEntry) {
	s.DeviceClass = getDeviceClass(info.DisplayUnit)
	// device classess for MQTT Number are somewhat limited currently
	if s.DeviceClass != "temperature" {
		s.DeviceClass = ""
	}
	s.CommandTopic = s.StateTopic + "/set"
	s.UnitOfMeasurement = info.DisplayUnit
	s.Min = info.Min
	s.Max = info.Max
	s.Step = info.Step
	s.Mode = "box"

	s.entityType = "number"
}

func (s *mqttCommon) marshal() (topic string, data []byte, err error) {
	topic = fmt.Sprintf("homeassistant/%s/%s/%s/config", s.entityType, s.deviceID, s.sensorName)
	data, err = json.Marshal(s)

	return topic, data, err
}

// PublishDiscoveryTopics publishes Home Assistant discovery topics for a device
func (m MQTT) PublishDiscoveryTopics(allTopics *topics.TopicData) {
	log.Printf("Publishing Home Assistant %s discovery topics...", allTopics.Kind())
	for _, value := range allTopics.GetAll() {

		var mqttAdvert mqttCommon
		mqttAdvert.encodeCommon(value, m.statusTopic(value.SensorName, value.Kind()), m.willTopic, allTopics.DeviceName())

		if value.EncodeFunction != "" {
			// Read-Write value
			if len(value.Values) == 0 {
				mqttAdvert.encodeNumber(value)
			} else if len(value.Values) > 2 || !(value.Values[0] == "Off" || value.Values[0] == "Disabled" || value.Values[0] == "Inactive") {
				mqttAdvert.encodeSelect(value)
			} else if len(value.Values) == 2 {
				mqttAdvert.encodeSwitch(value)
			} else {
				log.Println("Warning: Don't know how to encode " + value.SensorName)
			}
		} else {
			// Read only value
			if len(value.Values) == 2 && (value.Values[0] == "Off" || value.Values[0] == "Disabled" || value.Values[0] == "Inactive") {
				mqttAdvert.encodeBinarySensor(value)
			} else {
				mqttAdvert.encodeSensor(value)
			}
		}

		topic, data, err := mqttAdvert.marshal()
		if err != nil {
			log.Print(err)
			continue
		}

		m.Publish(topic, data, 0)
	}
	log.Println("Publishing Home Assistant discovery topics done.")
}
