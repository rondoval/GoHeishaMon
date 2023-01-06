package topics

import (
	"io/ioutil"
	"log"
	"sync"

	"gopkg.in/yaml.v2"
)

// DeviceType is an enum used for distinguishing between two emulated devices,
// i.e. the Optional PCB and an IoT gateway.
type DeviceType string

const (
	// Main - DeviceType = IoT device
	Main DeviceType = "main"
	// Optional - DeviceType = Optional PCB
	Optional = "optional"
)

type MappingEntry struct {
	Id   []byte `yaml:"id"`
	Name string `yaml:"name"`
}

type CodecEntry struct {
	EncodeFunction string `yaml:"encodeFunction"`
	DecodeFunction string `yaml:"decodeFunction"`
	Offset         int    `yaml:"offset"`
}

// TopicEntry represents a single entity, e.g. a sensor or configuration option.
type TopicEntry struct {
	SensorName  string         `yaml:"sensorName"`
	Codec       []CodecEntry   `yaml:"codec"`
	DisplayUnit string         `yaml:"displayUnit"`
	Category    string         `yaml:"category"`
	Values      []string       `yaml:"values"`
	Mapping     []MappingEntry `yaml:"mapping"`
	Min         float64        `yaml:"min"`
	Max         float64        `yaml:"max"`
	Step        float64        `yaml:"step"`

	currentValue      string
	currentValueMutex sync.Mutex
	kind              DeviceType
	writable          bool
}

func (t *TopicEntry) Writable() bool {
	return t.writable
}

// Kind returns the type of the device this TopicEntry is used with.
func (t *TopicEntry) Kind() DeviceType {
	return t.kind
}

// CurrentValue returns the current value of the entity, i.e. either received from the device or requested via MQTT.
// Thread safe.
func (t *TopicEntry) CurrentValue() string {
	t.currentValueMutex.Lock()
	defer t.currentValueMutex.Unlock()
	return t.currentValue
}

// UpdateValue updates the value of the entity.
// Returns true if the value has changed.
// Thread safe.
func (t *TopicEntry) UpdateValue(newValue string) bool {
	t.currentValueMutex.Lock()
	defer t.currentValueMutex.Unlock()
	if newValue != t.currentValue {
		t.currentValue = newValue
		return true
	}
	return false
}

// TopicData stores entities for a single device
type TopicData struct {
	allTopics       []*TopicEntry
	topicNameLookup map[string]*TopicEntry
	deviceName      string
	kind            DeviceType
}

// LoadTopics creates a TopicData strucutre by reading a YAML file.
// filename - name of the file to load
// deviceName - Name of the device, as should be used by HA discovery mechanism
// kind - either Main or Optional
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
		val.writable = false
		for _, codec := range val.Codec {
			if codec.EncodeFunction != "" {
				val.writable = true
			}
		}
	}

	t.deviceName = deviceName
	t.kind = kind
	log.Printf("Topic data loaded. %d entries.", len(t.allTopics))
	return &t
}

// Marshal stores the Optional PCB state to a file.
// Stores values that are being send to the heat pump only.
func (t *TopicData) Marshal(filename string) {
	m := make(map[string]string)
	for _, val := range t.allTopics {
		// we'll marshal only the values that we write/send to the pump
		// this is the state that is to be restored after reboot
		if val.Writable() {
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

// Unmarshal restores the Optional PCB state from a file.
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
		if val, ok := m[sensor.SensorName]; ok && val != "" {
			sensor.UpdateValue(val)
			changed = append(changed, sensor)
		}
	}
	return
}

// Lookup returns an entity with a name given as an argument.
func (t *TopicData) Lookup(name string) (*TopicEntry, bool) {
	elem, ok := t.topicNameLookup[name]
	return elem, ok
}

// GetAll returns all entities.
func (t *TopicData) GetAll() []*TopicEntry {
	return t.allTopics
}

// DeviceName returns  the device name as used on HA.
func (t TopicData) DeviceName() string {
	return t.deviceName
}

// Kind returns the type of the device, i.e. Main or Optional.
func (t TopicData) Kind() DeviceType {
	return t.kind
}
