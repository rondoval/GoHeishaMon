package main

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

var allTopics []topicData
var topicNameLookup map[string]topicData

type topicData struct {
	SensorName     string   `yaml:"sensorName"`
	DecodeFunction string   `yaml:"decodeFunction"`
	EncodeFunction string   `yaml:"encodeFunction"`
	DecodeOffset   int      `yaml:"decodeOffset"`
	DisplayUnit    string   `yaml:"displayUnit"`
	Category       string   `yaml:"category"`
	Values         []string `yaml:"values"`
	Min            int      `yaml:"min"`
	Max            int      `yaml:"max"`
	Step           int      `yaml:"step"`
	currentValue   string
}

func loadTopics() {
	log.Print("Loading topic data...")

	data, err := ioutil.ReadFile(config.topicsFile)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(data, &allTopics)
	if err != nil {
		log.Fatal(err)
	}

	topicNameLookup = make(map[string]topicData)
	for _, val := range allTopics {
		topicNameLookup[val.SensorName] = val
	}
	log.Print("Topic data loaded.")
}
