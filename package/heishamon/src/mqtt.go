package main

import (
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

func mqttPublish(mclient mqtt.Client, topic string, data interface{}, qos byte) {
	token := mclient.Publish(topic, qos, true, data)
	if token.Wait() && token.Error() != nil {
		log.Printf("Failed to publish, %v", token.Error())
	}
}

func onMQTTConnect(mclient mqtt.Client) {
	mqttPublish(mclient, config.mqttWillTopic, "online", 0)
	if config.ListenOnly == false {
		mclient.Subscribe(config.getStatusTopic("+/set", Main), 0, onAquareaCommand)
		if config.OptionalPCB == true {
			mclient.Subscribe(config.getStatusTopic("+/set", Optional), 0, onAquareaCommand)
		}
	}
	log.Print("MQTT connected")
}

func makeMQTTConn() mqtt.Client {
	log.Print("Setting up MQTT...")
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s://%s:%v", "tcp", config.MqttServer, config.MqttPort))
	opts.SetPassword(config.MqttPass)
	opts.SetUsername(config.MqttLogin)
	opts.SetClientID("GoHeishaMon-pub")
	opts.SetWill(config.mqttWillTopic, "offline", 0, true)
	opts.SetKeepAlive(time.Duration(config.MqttKeepalive) * time.Second)

	opts.SetCleanSession(true)  // don't want to receive entire backlog of setting changes
	opts.SetAutoReconnect(true) // default, but I want it explicit
	opts.SetConnectRetry(true)
	opts.SetOnConnectHandler(onMQTTConnect)

	// connect to broker
	client := mqtt.NewClient(opts)

	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		log.Fatalf("Fail to connect broker, %v", token.Error())
		//should not happen - SetConnectRetry=true
	}
	log.Println("MQTT set up completed")
	return client
}
