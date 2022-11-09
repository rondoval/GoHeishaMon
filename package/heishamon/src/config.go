package main

import (
	"io/ioutil"
	"log"
	"os"
	"path"

	"github.com/rondoval/GoHeishaMon/topics"
	"gopkg.in/yaml.v2"
)

type configStruct struct {
	DeviceName            string `yaml:"deviceName"`            // for HA discovery
	SerialPort            string `yaml:"serialPort"`            // serial port
	SerialTimeout         int    `yaml:"serialTimeout"`         // serial port timeout (ms)
	QueryInterval         int    `yaml:"queryInterval"`         // HP query interval (sec)
	ListenOnly            bool   `yaml:"listenOnly"`            // no commands at all
	OptionalPCB           bool   `yaml:"optionalPCB"`           // enable optional PCB emulation
	OptionalQueryInterval int    `yaml:"optionalQueryInterval"` // Optional PCB query interval (sec)
	OptionalSaveInterval  int    `yaml:"optionalSaveInterval"`  // Optional PCB data save interval (min)

	MqttServer     string `yaml:"mqttServer"`
	MqttPort       int    `yaml:"mqttPort"`
	MqttLogin      string `yaml:"mqttLogin"`
	MqttPass       string `yaml:"mqttPass"`
	MqttKeepalive  int    `yaml:"mqttKeepalive"`
	MqttTopicBase  string `yaml:"mqttTopicBase"`
	HAAutoDiscover bool   `yaml:"haAutoDiscover"`

	LogMqtt    bool `yaml:"logmqtt"`
	LogHexDump bool `yaml:"loghex"`
	LogDebug   bool `yaml:"logdebug"`

	topicsFile            string
	topicsOptionalPCBFile string
	optionalPCBFile       string
}

func (c configStruct) getDeviceName(kind topics.DeviceType) string {
	switch kind {
	case topics.Main:
		return c.DeviceName
	case topics.Optional:
		return c.DeviceName + " Optional PCB"
	default:
		return c.DeviceName
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

	c.topicsFile = path.Join(configPath, "topics.yaml")
	c.topicsOptionalPCBFile = path.Join(configPath, "topicsOptionalPCB.yaml")
	c.optionalPCBFile = path.Join(configPath, "optionalpcb.yaml")

	log.Println("Config file loaded")
}
