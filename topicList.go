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
	TopicName          string `yaml:"topicName"`
	TopicType          string `yaml:"topicType"`
	TopicBit           int    `yaml:"topicBit"`
	TopicFunction      string `yaml:"topicFunction"`
	TopicDisplayUnit   string `yaml:"topicDisplayUnit"`
	TopicValueTemplate string `yaml:"topicValueTemplate"`
	TopicValue         string
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
