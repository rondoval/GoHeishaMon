package main

import (
	"fmt"
	"log"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"go.bug.st/serial"
)

var panasonicQuery []byte = []byte{0x71, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

const panasonicQuerySize int = 110

var mqttKeepalive time.Duration
var config configStruct
var serialPort serial.Port
var switchTopics map[string]autoDiscoverStruct
var commandsChannel chan []byte

type commandStruct struct {
	value  [128]byte
	length int
}

func main() {
	config = readConfig()

	switchTopics = make(map[string]autoDiscoverStruct)
	c1 := make(chan bool, 1)
	commandsChannel = make(chan []byte, 100)
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
	mqttKeepalive = time.Second * time.Duration(config.MqttKeepalive)
	MC, MT := makeMQTTConn()
	if config.HAAutoDiscover == true {
		publishTopicsToAutoDiscover(MC, MT)
	}
	for {
		if MC.IsConnected() != true {
			MC, MT = makeMQTTConn()
		}
		var queueLen = len(commandsChannel)
		if queueLen > 50 {
			log.Println("Command queue length: ", len(commandsChannel))
		}

		select {
		case value := <-commandsChannel:
			sendCommand(value, len(value))
			time.Sleep(time.Second * time.Duration(config.SleepAfterCommand))

		default:
			sendCommand(panasonicQuery, panasonicQuerySize)
		}

		go func() {
			tbool := readSerial(MC, MT)
			c1 <- tbool
		}()

		select {
		case res := <-c1:
			log.Println("read ma status", res)
		case <-time.After(5 * time.Second):
			log.Println("out of time for read :(")
		}

		time.Sleep(PoolInterval)

	}

}

func makeMQTTConn() (mqtt.Client, mqtt.Token) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s://%s:%s", "tcp", config.MqttServer, config.MqttPort))
	opts.SetPassword(config.MqttPass)
	opts.SetUsername(config.MqttLogin)
	opts.SetClientID(config.MqttClientID)
	opts.SetWill(config.MqttSetBase+"/LWT", "Offline", 1, true)
	opts.SetKeepAlive(mqttKeepalive)
	opts.SetOnConnectHandler(startsub)
	opts.SetConnectionLostHandler(connLostHandler)

	// connect to broker
	client := mqtt.NewClient(opts)
	//defer client.Disconnect(uint(2))

	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		log.Printf("Fail to connect broker, %v", token.Error())
	}
	return client, token
}

func connLostHandler(c mqtt.Client, err error) {
	log.Printf("Connection lost, reason: %v\n", err)

	//Perform additional action...
}
