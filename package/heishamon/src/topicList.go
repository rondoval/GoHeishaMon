package main

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

type topicEntry struct {
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

	currentValue string
}

type topicData struct {
	allTopics       []topicEntry
	topicNameLookup map[string]topicEntry
	kind            DeviceType
}

func (t *topicData) loadTopics(filename string, kind DeviceType) {
	log.Print("Loading topic data from: ", filename)

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(data, &t.allTopics)
	if err != nil {
		log.Fatal(err)
	}

	t.topicNameLookup = make(map[string]topicEntry)
	for _, val := range t.allTopics {
		t.topicNameLookup[val.SensorName] = val
	}

	t.kind = kind
	log.Print("Topic data loaded.")
}

func (t *topicData) lookup(name string) (topicEntry, bool) {
	elem, ok := t.topicNameLookup[name]
	return elem, ok
}

func (t *topicData) getAll() []topicEntry {
	return t.allTopics
}

func (t *topicData) set(topic int, value string) {
	t.allTopics[topic].currentValue = value
}
