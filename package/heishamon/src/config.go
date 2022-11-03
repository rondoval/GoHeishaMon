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
	DeviceName            string `yaml:"deviceName"`            // for HA discovery
	SerialPort            string `yaml:"serialPort"`            // serial port
	QueryInterval         int    `yaml:"queryInterval"`         // HP query interval (sec)
	ListenOnly            bool   `yaml:"listenOnly"`            // no commands at all
	OptionalPCB           bool   `yaml:"optionalPCB"`           // enable optional PCB emulation
	OptionalQueryInterval int    `yaml:"optionalQueryInterval"` // Optional PCB query interval (sec)
	OptionalSaveInterval  int    `yaml:"optionalSaveInterval"`  // Optional PCB data save interval (min)

	MqttServer     string `yaml:"mqttServer"`
	MqttPort       string `yaml:"mqttPort"`
	MqttLogin      string `yaml:"mqttLogin"`
	MqttPass       string `yaml:"mqttPass"`
	MqttKeepalive  int    `yaml:"mqttKeepalive"`
	MqttTopicBase  string `yaml:"mqttTopicBase"`
	HAAutoDiscover bool   `yaml:"haAutoDiscover"`

	LogMqtt    bool `yaml:"logmqtt"`
	LogHexDump bool `yaml:"loghex"`

	mqttWillTopic string
	mqttLogTopic  string

	topicsFile            string
	topicsOptionalPCBFile string
	optionalPCBFile       string
	serialTimeout         time.Duration
}

const (
	Main = iota
	Optional
)
const main_topic = "main"
const optional_topic = "optional"

type DeviceType int

func (c configStruct) getDeviceName(kind DeviceType) string {
	switch kind {
	case Main:
		return c.DeviceName
	case Optional:
		return c.DeviceName + " Optional PCB"
	default:
		return c.DeviceName
	}
}

func (c configStruct) getStatusTopic(name string, kind DeviceType) string {
	switch kind {
	case Main:
		return fmt.Sprintf("%s/%s/%s", c.MqttTopicBase, main_topic, name)
	case Optional:
		return fmt.Sprintf("%s/%s/%s", c.MqttTopicBase, optional_topic, name)
	default:
		return c.MqttTopicBase
	}
}

func (c *configStruct) readConfig(configPath string) {
	var configFile = path.Join(configPath, "config.yaml")

	_, err := os.Stat(configFile)
	if err != nil {
		log.Fatalf("Config file is missing: %s ", configFile)
	}

	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(data, c)
	if err != nil {
		log.Fatal(err)
	}

	c.mqttWillTopic = c.MqttTopicBase + "/LWT"
	c.mqttLogTopic = c.MqttTopicBase + "/log"
	c.topicsFile = path.Join(configPath, "topics.yaml")
	c.topicsOptionalPCBFile = path.Join(configPath, "topicsOptionalPCB.yaml")
	c.optionalPCBFile = path.Join(configPath, "optionalpcb.raw")
	c.serialTimeout = 2 * time.Second

	log.Println("Config file loaded")
}
