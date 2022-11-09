package topics

import (
	"io/ioutil"
	"log"
	"sync"

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

	currentValue      string
	currentValueMutex sync.Mutex
	kind              DeviceType
}

func (t *TopicEntry) Kind() DeviceType {
	return t.kind
}

func (t *TopicEntry) CurrentValue() string {
	t.currentValueMutex.Lock()
	defer t.currentValueMutex.Unlock()
	return t.currentValue
}

func (t *TopicEntry) UpdateValue(newValue string) bool {
	t.currentValueMutex.Lock()
	defer t.currentValueMutex.Unlock()
	if newValue != t.currentValue {
		t.currentValue = newValue
		return true
	}
	return false
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
		t.topicNameLookup[val.SensorName] = val
		val.kind = kind
	}

	t.deviceName = deviceName
	t.kind = kind
	log.Printf("Topic data loaded. %d entries.", len(t.allTopics))
	return &t
}

func (t *TopicData) Marshal(filename string) {
	m := make(map[string]string)
	for _, val := range t.allTopics {
		// we'll marshal only the values that we write/send to the pump
		// this is the state that is to be restored after reboot
		if val.EncodeFunction != "" {
			m[val.SensorName] = val.CurrentValue()
		}
	}

	data, err := yaml.Marshal(m)
	if err != nil {
		log.Printf("Error while marshalling optional PCB state: %v", err)
		return
	}

	err = ioutil.WriteFile(filename, data, 0644)
	if err != nil {
		log.Printf("Error while saving optional PCB state: %v", err)
		return
	}
}

func (t *TopicData) Unmarshal(filename string) (changed []*TopicEntry) {
	changed = make([]*TopicEntry, 0, len(t.allTopics))

	data, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Printf("Error while loading optional PCB state: %v", err)
		return
	}

	var m map[string]string
	err = yaml.Unmarshal(data, &m)
	if err != nil {
		log.Printf("Error while unmarshalling optional PCB state: %v", err)
		return
	}

	for _, sensor := range t.allTopics {
		if val, ok := m[sensor.SensorName]; ok {
			if val != "" {
				sensor.UpdateValue(val)
				changed = append(changed, sensor)
			}
		}
	}
	return
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
