package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func publishTopicsToAutoDiscover(mclient mqtt.Client) {
	for k, v := range allTopics {
		var m autoDiscoverStruct
		m.UID = fmt.Sprintf("Aquarea-%s-%d", config.MqttLogin, k)
		if v.SensorType == "" {
			v.SensorType = "sensor"
		}
		//m.ValueTemplate = v.TopicValueTemplate

		m.UnitOfM = v.DisplayUnit

		if v.SensorType == "binary_sensor" {
			m.UnitOfM = ""
			m.PayloadOn = "1"
			m.PayloadOff = "0"
			m.ValueTemplate = `{{ value }}`
		}
		if v.DisplayUnit == "°C" {
			m.DeviceClass = "temperature"
		}
		if v.DisplayUnit == "W" {
			m.DeviceClass = "power"
		}
		m.StateTopic = fmt.Sprintf("%s/%s", config.mqttValuesTopic, v.SensorName)
		m.Name = fmt.Sprintf("TEST-%s", v.SensorName)
		topicValue, err := json.Marshal(m)
		if err != nil {
			log.Println(err)
		}
		//v.TopicType = "sensor"

		TOP := fmt.Sprintf("%s/%s/%s/config", config.mqttDiscoveryTopic, v.SensorType, strings.ReplaceAll(m.Name, " ", "_"))
		token := mclient.Publish(TOP, byte(0), true, topicValue)
		if token.Wait() && token.Error() != nil {
			log.Printf("Fail to publish, %v", token.Error())
		}

	}

	for _, vs := range switchTopics {
		if vs.ValueTemplate == "" {
			vs.PayloadOff = "0"
			vs.PayloadOn = "1"
		}
		vs.Optimistic = "true"
		topicValue, err := json.Marshal(vs)
		if err != nil {
			log.Println(err)
		}

		TOP := fmt.Sprintf("%s/%s/%s/config", config.mqttDiscoveryTopic, "switch", strings.ReplaceAll(vs.Name, " ", "_"))
		token := mclient.Publish(TOP, byte(0), true, topicValue)
		if token.Wait() && token.Error() != nil {
			log.Printf("Fail to publish, %v", token.Error())
		}
	}

}

type autoDiscoverStruct struct {
	DeviceClass   string `json:"device_class,omitempty"`
	Name          string `json:"name,omitempty"`
	StateTopic    string `json:"state_topic,omitempty"`
	UnitOfM       string `json:"unit_of_measurement,omitempty"`
	ValueTemplate string `json:"value_template,omitempty"`
	CommandTopic  string `json:"command_topic,omitempty"`
	UID           string `json:"unique_id,omitempty"`
	PayloadOn     string `json:"payload_on,omitempty"`
	PayloadOff    string `json:"payload_off,omitempty"`
	Optimistic    string `json:"optimistic,omitempty"`
	StateON       string `json:"state_on,omitempty"`
	StateOff      string `json:"state_off,omitempty"`
}
