package main

import (
	"io/ioutil"
	"log"
	"runtime"

	"gopkg.in/yaml.v2"
)

const topicsFileOther = "/etc/gh/topics.yaml"
const topicsFileWindows = "topics.yaml"

var allTopics []topicData

type topicData struct {
	SensorName     string   `yaml:"sensorName"`
	SensorType     string   `yaml:"sensorType"`
	DecodeOffset   int      `yaml:"decodeOffset"`
	DecodeFunction string   `yaml:"decodeFunction"`
	DisplayUnit    string   `yaml:"displayUnit"`
	Values         []string `yaml:"values"`
	currentValue   string
}

func loadTopics() {
	var topicFile string
	if runtime.GOOS == "windows" {
		topicFile = topicsFileWindows
	} else {
		topicFile = topicsFileOther
	}

	data, err := ioutil.ReadFile(topicFile)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(data, &allTopics)
	if err != nil {
		log.Fatal(err)
	}
}
