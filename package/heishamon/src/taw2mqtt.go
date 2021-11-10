package main

import (
	"flag"
	"io/ioutil"
	"log"
	"time"

	"github.com/tarm/serial"
)

var panasonicQuery []byte = []byte{0x71, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
var optionalPCBQuery []byte = []byte{0xF1, 0x11, 0x01, 0x50, 0x00, 0x00, 0x40, 0xFF, 0xFF, 0xE5, 0xFF, 0xFF, 0x00, 0xFF, 0xEB, 0xFF, 0xFF, 0x00, 0x00}
var config configStruct

var serialPort *serial.Port
var commandsChannel chan []byte

func main() {
	var configPath = flag.String("path", "/etc/heishamon", "Path to Heishamon configuration files")
	flag.Parse()

	redirectLogSyslog()
	log.SetFlags(log.Lshortfile)
	log.Println("GoHeishaMon loading...")
	config = readConfig(*configPath)

	serialConfig := &serial.Config{Name: config.Device, Baud: 9600, Parity: serial.ParityEven, StopBits: serial.Stop1, ReadTimeout: config.serialTimeout}
	var err error
	serialPort, err = serial.OpenPort(serialConfig)
	if err != nil {
		// no point in continuing, no config
		log.Fatal(err)
	}
	log.Print("Serial port open")

	commandsChannel = make(chan []byte, 100)
	loadTopics()
	if config.OptionalPCB {
		loadOptionalPCB()
	}

	mclient := makeMQTTConn()
	redirectLogMQTT(mclient)
	if config.HAAutoDiscover == true {
		publishDiscoveryTopics(mclient)
	}

	queryTicker := time.NewTicker(time.Second * time.Duration(config.ReadInterval))
	optionPCBSaveTicker := time.NewTicker(config.optionalPCBSaveTime)
	log.Print("Entering main loop")
	sendCommand(panasonicQuery)
	for {
		time.Sleep(config.serialTimeout)
		readSerial(mclient)

		var queueLen = len(commandsChannel)
		if queueLen > 10 {
			log.Print("Command queue length: ", len(commandsChannel))
		}

		select {
		case <-optionPCBSaveTicker.C:
			if config.OptionalPCB {
				saveOptionalPCB()
			}

		case value := <-commandsChannel:
			sendCommand(value)

		case <-queryTicker.C:
			commandsChannel <- panasonicQuery

		default:
			if config.OptionalPCB == true && config.ListenOnly == false {
				commandsChannel <- optionalPCBQuery
			}
		}
	}
}

func saveOptionalPCB() {
	err := ioutil.WriteFile(config.optionalPCBFile, optionalPCBQuery, 0644)
	if err != nil {
		log.Print(err)
	} else {
		log.Print("Optional PCB data stored")
	}
}

func loadOptionalPCB() {
	data, err := ioutil.ReadFile(config.optionalPCBFile)
	if err != nil {
		log.Print(err)
	} else {
		optionalPCBQuery = data
		log.Print("Optional PCB data loaded")
	}

}
