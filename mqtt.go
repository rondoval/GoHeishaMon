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
		log.Printf("Fail to publish, %v", token.Error())
	}
}

func onMQTTConnect(mclient mqtt.Client) {
	mqttPublish(mclient, config.mqttWillTopic, "online", 0)
	if config.ListenOnly == false {
		mclient.Subscribe(getCommandTopic("+"), 2, onCommand)
	}
	//TODO  shall we re post all data?
}

func makeMQTTConn() mqtt.Client {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s://%s:%v", "tcp", config.MqttServer, config.MqttPort))
	opts.SetPassword(config.MqttPass)
	opts.SetUsername(config.MqttLogin)
	opts.SetClientID("GoHeishaMon-pub")
	opts.SetWill(config.mqttWillTopic, "offline", 1, true)
	opts.SetKeepAlive(time.Second * time.Duration(config.MqttKeepalive))

	opts.SetCleanSession(true)  // don't want to receive entire backlog of setting changes
	opts.SetAutoReconnect(true) // default, but I want it explicit
	opts.SetOnConnectHandler(onMQTTConnect)

	// connect to broker
	client := mqtt.NewClient(opts)

	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		log.Fatalf("Fail to connect broker, %v", token.Error())
		//TODO should restart/retry somehow... indefinitely
	}
	return client
}
