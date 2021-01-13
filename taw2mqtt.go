package main

import (
	"log"
	"time"

	"github.com/tarm/serial"
)

var panasonicQuery []byte = []byte{0x71, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
var optionalPCBQuery []byte = []byte{0xF1, 0x11, 0x01, 0x50, 0x00, 0x00, 0x40, 0xFF, 0xFF, 0xE5, 0xFF, 0xFF, 0x00, 0xFF, 0xEB, 0xFF, 0xFF, 0x00, 0x00}
var config configStruct

//var serialPort serial.Port
var serialPort *serial.Port
var commandsChannel chan []byte

func main() {
	config = readConfig()

	commandsChannel = make(chan []byte, 100)
	timeoutChannel := make(chan bool)

	go updateConfigLoop()

	// ports, err := serial.GetPortsList()
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// if len(ports) == 0 {
	// 	log.Fatal("No serial ports found!")
	// }
	// for _, port := range ports {
	// 	log.Printf("Found port: %v\n", port)
	// }
	serialConfig := &serial.Config{Name: config.Device, Baud: 9600, Parity: serial.ParityEven, StopBits: serial.Stop1, ReadTimeout: 5 * time.Second}
	var err error
	serialPort, err = serial.OpenPort(serialConfig)
	// mode := &serial.Mode{
	// 	BaudRate: 9600,
	// 	Parity:   serial.EvenParity,
	// 	DataBits: 8,
	// 	StopBits: serial.OneStopBit,
	// }
	// serialPort, err = serial.Open(config.Device, mode)
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
