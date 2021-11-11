package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"time"

	"gopkg.in/yaml.v2"
)

type configStruct struct {
	DeviceName      string `yaml:"deviceName"`      // for HA discovery
	Device          string `yaml:"device"`          // serial port
	ReadInterval    int    `yaml:"readInterval"`    // HP query interval
	ListenOnly      bool   `yaml:"listenOnly"`      // no commands at all
	OptionalPCB     bool   `yaml:"optionalPCB"`     // enable optional PCB emulation
	EnableOSCommand bool   `yaml:"enableOSCommand"` // enable OS commands

	MqttServer     string `yaml:"mqttServer"`
	MqttPort       string `yaml:"mqttPort"`
	MqttLogin      string `yaml:"mqttLogin"`
	MqttPass       string `yaml:"mqttPass"`
	MqttKeepalive  int    `yaml:"mqttKeepalive"`
	MqttTopicBase  string `yaml:"mqttTopicBase"`
	HAAutoDiscover bool   `yaml:"haAutoDiscover"`

	LogMqtt    bool `yaml:"logmqtt"`
	LogHexDump bool `yaml:"loghex"`

	//topics
	mqttWillTopic      string
	mqttLogTopic       string
	mqttValuesTopic    string
	mqttPcbValuesTopic string
	mqttCommandsTopic  string

	topicsFile          string
	optionalPCBFile     string
	serialTimeout       time.Duration
	optionalPCBSaveTime time.Duration
}

func getStatusTopic(name string) string {
	return fmt.Sprintf("%s/%s", config.mqttValuesTopic, name)
}

func getCommandTopic(name string) string {
	return fmt.Sprintf("%s/%s", config.mqttCommandsTopic, name)
}

func getPcbStatusTopic(name string) string {
	return fmt.Sprintf("%s/%s", config.mqttPcbValuesTopic, name)
}

func readConfig(configPath string) configStruct {
	var configFile = path.Join(configPath, "config.yaml")

	_, err := os.Stat(configFile)
	if err != nil {
		log.Fatalf("Config file is missing: %s ", configFile)
	}

	var config configStruct

	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatal(err)
	}

	config.mqttWillTopic = config.MqttTopicBase + "/LWT"
	config.mqttLogTopic = config.MqttTopicBase + "/log"
	config.mqttValuesTopic = config.MqttTopicBase + "/main"
	config.mqttPcbValuesTopic = config.MqttTopicBase + "/optional"
	config.mqttCommandsTopic = config.MqttTopicBase + "/commands"
	config.optionalPCBFile = path.Join(configPath, "optionalpcb.raw")
	config.topicsFile = path.Join(configPath, "topics.yaml")
	config.serialTimeout = 2 * time.Second
	config.optionalPCBSaveTime = 5 * time.Minute

	log.Println("Config file loaded")

	return config
}
