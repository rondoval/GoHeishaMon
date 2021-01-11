package main

import (
	"io/ioutil"
	"log"
	"runtime"

	"gopkg.in/yaml.v2"
)

const topicsFileOther = "/data/topics.yaml"
const topicsFileWindows = "topics.yaml"

var allTopics []topicData

type topicData struct {
	TopicNumber        int    `yaml:"topicNumber"`
	TopicName          string `yaml:"topicName"`
	TopicType          string `yaml:"topicType"`
	TopicBit           int    `yaml:"topicBit"`
	TopicFunction      string `yaml:"topicFunction"`
	TopicUnit          string `yaml:"topicUnit"`
	TopicA2M           string `yaml:"topicA2M"`
	TopicDisplayUnit   string `yaml:"topicDisplayUnit"`
	TopicValueTemplate string `yaml:"topicValueTemplate"`
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
	actData = make([]string, len(allTopics))
}
