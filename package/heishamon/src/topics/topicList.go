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
	sensorName     string   `yaml:"sensorName"`
	decodeFunction string   `yaml:"decodeFunction"`
	encodeFunction string   `yaml:"encodeFunction"`
	decodeOffset   int      `yaml:"decodeOffset"`
	displayUnit    string   `yaml:"displayUnit"`
	category       string   `yaml:"category"`
	values         []string `yaml:"values"`
	min            int      `yaml:"min"`
	max            int      `yaml:"max"`
	step           int      `yaml:"step"`

	CurrentValue string
	kind         DeviceType
}

func (t TopicEntry) SensorName() string {
	return t.sensorName
}

func (t TopicEntry) DecodeFunction() string {
	return t.decodeFunction
}

func (t TopicEntry) EncodeFunction() string {
	return t.encodeFunction
}

func (t TopicEntry) DecodeOffset() int {
	return t.decodeOffset
}

func (t TopicEntry) DisplayUnit() string {
	return t.displayUnit
}

func (t TopicEntry) Category() string {
	return t.category
}

func (t TopicEntry) Values() []string {
	return t.values
}

func (t TopicEntry) Min() int {
	return t.min
}

func (t TopicEntry) Max() int {
	return t.max
}

func (t TopicEntry) Step() int {
	return t.step
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

	err = yaml.Unmarshal(data, &t.allTopics)
	if err != nil {
		log.Fatal(err)
	}

	t.topicNameLookup = make(map[string]*TopicEntry)
	for _, val := range t.allTopics {
		t.topicNameLookup[val.sensorName] = val
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
