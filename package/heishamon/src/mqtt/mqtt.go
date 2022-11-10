package mqtt

import (
	"fmt"
	"log"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/rondoval/GoHeishaMon/topics"
)

type MQTT struct {
	mclient   paho.Client
	baseTopic string
	willTopic string

	commandChannel chan Command
}

type Command struct {
	Topic     string
	Payload   string
	AllTopics *topics.TopicData
}

func (m MQTT) Publish(topic string, data interface{}, qos byte) {
	token := m.mclient.Publish(topic, qos, true, data)
	go func() {
		if token.Wait() && token.Error() != nil {
			log.Printf("Failed to publish, %v", token.Error())
		}
	}()
}

func (m MQTT) PublishValue(value *topics.TopicEntry) {
	m.Publish(m.statusTopic(value.SensorName, value.Kind()), value.CurrentValue(), 0)

}

func (m MQTT) LogTopic() string {
	return m.baseTopic + "/log"
}

func (m MQTT) CommandChannel() chan Command {
	return m.commandChannel
}

func (m MQTT) statusTopic(name string, kind topics.DeviceType) string {
	return fmt.Sprintf("%s/%s/%s", m.baseTopic, string(kind), name)
}

type Options struct {
	Server         string
	Port           int
	Username       string
	Password       string
	BaseTopic      string
	KeepAlive      time.Duration
	ListenOnly     bool
	OptionalPCB    bool
	CommandTopics  *topics.TopicData
	OptionalTopics *topics.TopicData
}

func MakeMQTTConn(opt Options) MQTT {
	log.Print("Setting up MQTT...")
	var mqtt MQTT

	mqtt.commandChannel = make(chan Command, 20)
	mqtt.baseTopic = opt.BaseTopic
	mqtt.willTopic = opt.BaseTopic + "/LWT"

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
		if opt.ListenOnly == false {
			tokenMain := mclient.Subscribe(mqtt.statusTopic("+/set", topics.Main), 0, func(client paho.Client, payload paho.Message) {
				mqtt.commandChannel <- Command{Topic: payload.Topic(), Payload: string(payload.Payload()), AllTopics: opt.CommandTopics}
			})
			go func() {
				if tokenMain.Wait() && tokenMain.Error() != nil {
					log.Printf("Failed to subscribe, %v", tokenMain.Error())
				}
			}()
			if opt.OptionalPCB == true {
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
