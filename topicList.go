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
	DecodeFunction string   `yaml:"decodeFunction"`
	DecodeOffset   int      `yaml:"decodeOffset"`
	DisplayUnit    string   `yaml:"displayUnit"`
	Values         []string `yaml:"values"`
	Command        string   `yaml:"command"`
	currentValue   string
}

func loadTopics() {
	log.Print("Loading topic data...")
	var topicFile string
	if runtime.GOOS == "windows" {
		topicFile = topicsFileWindows
	} else {
		topicFile = topicsFileOther
	}

	data, err := ioutil.ReadFile(topicFile)
	if err != nil {
		logErrorPause(err)
	}

	err = yaml.Unmarshal(data, &allTopics)
	if err != nil {
		logErrorPause(err)
	}
	log.Println(" loaded.")
}
