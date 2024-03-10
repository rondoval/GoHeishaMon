// Package mqtt implements MQTT communication with Home Assistant.
package mqtt

import (
	"fmt"
	"log"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/rondoval/GoHeishaMon/topics"
)

// MQTT holds state of the MQTT client
type MQTT struct {
	mclient   paho.Client
	baseTopic string
	willTopic string

	commandChannel chan Command
}

// Command is the structure being passed on the channel returned by CommandChannel()
type Command struct {
	Topic     string
	Payload   string
	AllTopics *topics.TopicData
}

// Publish data via MQTT
func (m MQTT) Publish(topic string, data interface{}, qos byte) {
	token := m.mclient.Publish(topic, qos, true, data)
	go func() {
		if token.Wait() && token.Error() != nil {
			log.Printf("Failed to publish, %v", token.Error())
		}
	}()
}

// PublishValue posts an entity value via MQTT.
func (m MQTT) PublishValue(value *topics.TopicEntry) {
	m.Publish(m.statusTopic(value.SensorName, value.Kind()), value.CurrentValue(), 0)
}

// LogTopic returns a topic that shall be used by logging for posting log entries.
func (m MQTT) LogTopic() string {
	return m.baseTopic + "/log"
}

// CommandChannel returns a channel on which incoming MQTT requests are sent.
func (m MQTT) CommandChannel() chan Command {
	return m.commandChannel
}

func (m MQTT) statusTopic(name string, kind topics.DeviceType) string {
	return fmt.Sprintf("%s/%s/%s", m.baseTopic, string(kind), name)
}

// Options is used as an argument to MakeMQTTConn
type Options struct {
	Server         string            // MQTT server address
	Port           int               // MQTT server port
	Username       string            // MQTT server username
	Password       string            // MQTT server password
	BaseTopic      string            // Base topic for use on MQTT
	KeepAlive      time.Duration     // MQTT keepalive
	ListenOnly     bool              // Do not send anything to the heat pump
	OptionalPCB    bool              // Enable Optional PCB emulation
	CommandTopics  *topics.TopicData // Structure describing IoT device entities
	OptionalTopics *topics.TopicData // Structure describing Optional PCB entities
}

// MakeMQTTConn creates a new MQTT connection.
// Sets up subscriptions.
func MakeMQTTConn(opt Options) MQTT {
	log.Print("Setting up MQTT...")
	mqtt := MQTT{
		commandChannel: make(chan Command, 20),
		baseTopic:      opt.BaseTopic,
		willTopic:      opt.BaseTopic + "/LWT",
	}

	pahoOpt := paho.NewClientOptions()
	pahoOpt.AddBroker(fmt.Sprintf("%s://%s:%d", "tcp", opt.Server, opt.Port))
	pahoOpt.SetPassword(opt.Password)
	pahoOpt.SetUsername(opt.Username)
	pahoOpt.SetClientID("GoHeishaMon-pub")
	pahoOpt.SetWill(mqtt.willTopic, "offline", 0, true)
	pahoOpt.SetKeepAlive(opt.KeepAlive)

	pahoOpt.SetCleanSession(true)  // don't want to receive entire backlog of setting changes
	pahoOpt.SetAutoReconnect(true) // default, but I want it explicit
	pahoOpt.SetConnectRetry(true)
	pahoOpt.SetOnConnectHandler(func(mclient paho.Client) {
		mqtt.Publish(mqtt.willTopic, "online", 0)
		if !opt.ListenOnly {
			tokenMain := mclient.Subscribe(mqtt.statusTopic("+/set", topics.Main), 0, func(client paho.Client, payload paho.Message) {
				mqtt.commandChannel <- Command{Topic: payload.Topic(), Payload: string(payload.Payload()), AllTopics: opt.CommandTopics}
			})
			go func() {
				if tokenMain.Wait() && tokenMain.Error() != nil {
					log.Printf("Failed to subscribe, %v", tokenMain.Error())
				}
			}()
			if opt.OptionalPCB {
				tokenOptional := mclient.Subscribe(mqtt.statusTopic("+/set", topics.Optional), 0, func(client paho.Client, payload paho.Message) {
					mqtt.commandChannel <- Command{Topic: payload.Topic(), Payload: string(payload.Payload()), AllTopics: opt.OptionalTopics}
				})
				go func() {
					if tokenOptional.Wait() && tokenOptional.Error() != nil {
						log.Printf("Failed to subscribe, %v", tokenOptional.Error())
					}
				}()
			}
		}
		log.Print("MQTT connected")
	})

	// connect to broker
	mqtt.mclient = paho.NewClient(pahoOpt)

	token := mqtt.mclient.Connect()
	go func() {
		if token.Wait() && token.Error() != nil {
			log.Printf("Failed to connect broker, %v", token.Error())
			//should not happen - SetConnectRetry=true
		}
	}()
	log.Println("MQTT set up completed")
	return mqtt
}
