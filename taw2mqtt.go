package main

import (
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.bug.st/serial"
)

var panasonicQuery []byte = []byte{0x71, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
var optionalPCBQuery []byte = []byte{0xF1, 0x11, 0x01, 0x50, 0x00, 0x00, 0x40, 0xFF, 0xFF, 0xE5, 0xFF, 0xFF, 0x00, 0xFF, 0xEB, 0xFF, 0xFF, 0x00, 0x00}
var config configStruct
var serialPort serial.Port
var commandsChannel chan []byte

func main() {
	config = readConfig()

	commandsChannel = make(chan []byte, 100)
	timeoutChannel := make(chan bool)

	go updateConfigLoop()

	ports, err := serial.GetPortsList()
	if err != nil {
		log.Fatal(err)
	}
	if len(ports) == 0 {
		log.Fatal("No serial ports found!")
	}
	for _, port := range ports {
		log.Printf("Found port: %v\n", port)
	}
	mode := &serial.Mode{
		BaudRate: 9600,
		Parity:   serial.EvenParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}
	serialPort, err = serial.Open(config.Device, mode)
	if err != nil {
		log.Println(err)
	}

	PoolInterval := time.Second * time.Duration(config.ReadInterval)
	loadTopics()
	mclient := makeMQTTConn()
	if config.HAAutoDiscover == true {
		publishDiscoveryTopics(mclient)
	}

	for {
		var queueLen = len(commandsChannel)
		if queueLen > 50 {
			log.Println("Command queue length: ", len(commandsChannel))
		}

		select {
		case value := <-commandsChannel:
			sendCommand(value)
			time.Sleep(time.Second * time.Duration(config.SleepAfterCommand))

		default:
			sendCommand(panasonicQuery)
			//sendCommand()optionalPCBQuery
		}

		go func() {
			readSerial(mclient)
			timeoutChannel <- true
		}()

		select {
		case <-timeoutChannel:
			// serial read done
		case <-time.After(5 * time.Second):
			log.Println("Serial port read timeout")
		}

		time.Sleep(PoolInterval)
	}
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
	opts.SetOnConnectHandler(subscribe)

	// connect to broker
	client := mqtt.NewClient(opts)

	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		log.Fatalf("Fail to connect broker, %v", token.Error())
		//TODO should restart/retry somehow... indefinitely
	}
	return client
}
