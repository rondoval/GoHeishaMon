package main

import (
	"io/ioutil"
	"log"
	"time"

	"github.com/tarm/serial"
)

const serialTimeout = 2 * time.Second
const optionalPCBSaveTime = 5 * time.Minute
const optionalPCBFile = "/etc/gh/optionalpcb.raw"

var panasonicQuery []byte = []byte{0x71, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
var optionalPCBQuery []byte = []byte{0xF1, 0x11, 0x01, 0x50, 0x00, 0x00, 0x40, 0xFF, 0xFF, 0xE5, 0xFF, 0xFF, 0x00, 0xFF, 0xEB, 0xFF, 0xFF, 0x00, 0x00}
var config configStruct

var serialPort *serial.Port
var commandsChannel chan []byte

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	log.Println("GoHeishaMon loading...")
	go updateConfigLoop()
	config = readConfig()

	serialConfig := &serial.Config{Name: config.Device, Baud: 9600, Parity: serial.ParityEven, StopBits: serial.Stop1, ReadTimeout: serialTimeout}
	var err error
	serialPort, err = serial.OpenPort(serialConfig)
	if err != nil {
		// no point in continuing, wating for new config file
		logErrorPause(err)
	}
	log.Println("Serial port open")

	commandsChannel = make(chan []byte, 100)
	loadTopics()
	if config.OptionalPCB {
		loadOptionalPCB()
	}

	mclient := makeMQTTConn()
	redirectLog(mclient)
	if config.HAAutoDiscover == true {
		publishDiscoveryTopics(mclient)
	}

	queryTicker := time.NewTicker(time.Second * time.Duration(config.ReadInterval))
	optionPCBSaveTicker := time.NewTicker(optionalPCBSaveTime)
	log.Println("Entering main loop")
	sendCommand(panasonicQuery)
	for {
		time.Sleep(serialTimeout)
		readSerial(mclient)

		var queueLen = len(commandsChannel)
		if queueLen > 10 {
			log.Println("Command queue length: ", len(commandsChannel))
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
	err := ioutil.WriteFile(optionalPCBFile, optionalPCBQuery, 0644)
	if err != nil {
		log.Println(err)
	} else {
		log.Println("Optional PCB data stored")
	}
}

func loadOptionalPCB() {
	data, err := ioutil.ReadFile(optionalPCBFile)
	if err != nil {
		log.Println(err)
	} else {
		optionalPCBQuery = data
		log.Println("Optional PCB data loaded")
	}

}
