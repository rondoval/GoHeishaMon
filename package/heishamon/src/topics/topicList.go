package topics

import (
	"io/ioutil"
	"log"

	"gopkg.in/yaml.v2"
)

type DeviceType string

const (
	Main     DeviceType = "main"
	Optional            = "optional"
)

type TopicEntry struct {
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

	CurrentValue string
	kind         DeviceType
}

func (t TopicEntry) Kind() DeviceType {
	return t.kind
}

type TopicData struct {
	allTopics       []*TopicEntry
	topicNameLookup map[string]*TopicEntry
	deviceName      string
	kind            DeviceType
}

func LoadTopics(filename, deviceName string, kind DeviceType) *TopicData {
	log.Print("Loading topic data from: ", filename)
	var t TopicData

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(data, t.allTopics)
	if err != nil {
		log.Fatal(err)
	}

	t.topicNameLookup = make(map[string]*TopicEntry)
	for _, val := range t.allTopics {
		t.topicNameLookup[val.SensorName] = val
		val.kind = kind
	}

	t.deviceName = deviceName
	t.kind = kind
	log.Printf("Topic data loaded. %d entries.", len(t.allTopics))
	return &t
}

func (t *TopicData) Lookup(name string) (*TopicEntry, bool) {
	elem, ok := t.topicNameLookup[name]
	return elem, ok
}

func (t *TopicData) GetAll() []*TopicEntry {
	return t.allTopics
}

func (t TopicData) DeviceName() string {
	return t.deviceName
}

func (t TopicData) Kind() DeviceType {
	return t.kind
}
