package main

import (
	"encoding/json"
	"fmt"
	"strings"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func publishTopicsToAutoDiscover(mclient mqtt.Client, token mqtt.Token) {
	for k, v := range allTopics {
		var m autoDiscoverStruct
		m.UID = fmt.Sprintf("Aquarea-%s-%d", config.MqttLogin, k)
		if v.TopicType == "" {
			v.TopicType = "sensor"
		}
		m.ValueTemplate = v.TopicValueTemplate

		m.UnitOfM = v.TopicDisplayUnit

		if v.TopicType == "binary_sensor" {
			m.UnitOfM = ""
			m.PayloadOn = "1"
			m.PayloadOff = "0"
			m.ValueTemplate = `{{ value }}`
		}
		if v.TopicDisplayUnit == "°C" {
			m.DeviceClass = "temperature"
		}
		if v.TopicDisplayUnit == "W" {
			m.DeviceClass = "power"
		}
		m.StateTopic = fmt.Sprintf("%s/%s", config.MqttTopicBase, v.TopicName)
		m.Name = fmt.Sprintf("TEST-%s", v.TopicName)
		topicValue, err := json.Marshal(m)
		//v.TopicType = "sensor"

		fmt.Println(err)
		TOP := fmt.Sprintf("%s/%s/%s/config", config.MqttTopicBase, v.TopicType, strings.ReplaceAll(m.Name, " ", "_"))
		fmt.Println("Publikuje do ", TOP, "warosc", string(topicValue))
		token = mclient.Publish(TOP, byte(0), false, topicValue)
		if token.Wait() && token.Error() != nil {
			fmt.Printf("Fail to publish, %v", token.Error())
		}

	}

	for _, vs := range switchTopics {
		if vs.ValueTemplate == "" {
			vs.PayloadOff = "0"
			vs.PayloadOn = "1"
		}
		vs.Optimistic = "true"
		topicValue, err := json.Marshal(vs)

		fmt.Println(err)
		TOP := fmt.Sprintf("%s/%s/%s/config", config.MqttTopicBase, "switch", strings.ReplaceAll(vs.Name, " ", "_"))
		fmt.Println("Publikuje do ", TOP, "warosc", string(topicValue))
		token = mclient.Publish(TOP, byte(0), false, topicValue)
		if token.Wait() && token.Error() != nil {
			fmt.Printf("Fail to publish, %v", token.Error())
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
