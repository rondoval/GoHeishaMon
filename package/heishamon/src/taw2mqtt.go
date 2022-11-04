package main

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"time"

	"github.com/chmorgan/go-serial2/serial"
)

var panasonicQuery []byte = []byte{0x71, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
var optionalPCBQuery []byte = []byte{0xF1, 0x11, 0x01, 0x50, 0x00, 0x00, 0x40, 0xFF, 0xFF, 0xE5, 0xFF, 0xFF, 0x00, 0xFF, 0xEB, 0xFF, 0xFF, 0x00, 0x00}

var config configStruct
var commandTopics topicData
var optionalPCBTopics topicData

var serialPort io.ReadWriteCloser
var commandsChannel chan []byte

func main() {
	var configPath = flag.String("path", "/etc/heishamon", "Path to Heishamon configuration files")
	flag.Parse()

	redirectLogSyslog()
	log.SetFlags(log.Lshortfile)
	log.Println("GoHeishaMon loading...")
	config.readConfig(*configPath)

	options := serial.OpenOptions{
		PortName:              config.SerialPort,
		BaudRate:              9600,
		DataBits:              8,
		StopBits:              1,
		ParityMode:            serial.PARITY_EVEN,
		RTSCTSFlowControl:     false,
		InterCharacterTimeout: uint(config.serialTimeout),
		MinimumReadSize:       0,
	}
	var err error
	serialPort, err = serial.Open(options)
	if err != nil {
		// no point in continuing, no config
		log.Fatal(err)
	}
	log.Print("Serial port open")
	defer serialPort.Close()

	commandsChannel = make(chan []byte, 100)
	commandTopics.loadTopics(config.topicsFile, Main)
	optionalPCBTopics.loadTopics(config.topicsOptionalPCBFile, Optional)

	if config.OptionalPCB {
		loadOptionalPCB()
	}

	mclient := makeMQTTConn()
	redirectLogMQTT(mclient)
	if config.HAAutoDiscover == true {
		publishDiscoveryTopics(mclient, commandTopics, config)
		if config.OptionalPCB == true {
			publishDiscoveryTopics(mclient, optionalPCBTopics, config)
		}
	}

	queryTicker := time.NewTicker(time.Second * time.Duration(config.QueryInterval))
	optionPCBSaveTicker := time.NewTicker(time.Minute * time.Duration(config.OptionalSaveInterval))
	optionQueryTicker := time.NewTicker(time.Second * time.Duration(config.OptionalQueryInterval))

	log.Print("Entering main loop")
	if config.OptionalPCB == true {
		sendCommand(optionalPCBQuery)
	}
	sendCommand(panasonicQuery)

	for {
		var queueLen = len(commandsChannel)
		if queueLen > 10 {
			log.Print("Command queue length: ", queueLen)
		}

		select {
		case <-optionPCBSaveTicker.C:
			if config.OptionalPCB {
				saveOptionalPCB()
			}

		case value := <-commandsChannel:
			sendCommand(value)

		case <-optionQueryTicker.C:
			if config.OptionalPCB == true && config.ListenOnly == false {
				commandsChannel <- optionalPCBQuery
			}

		case <-queryTicker.C:
			commandsChannel <- panasonicQuery

		default:
			data := readSerial(config.LogHexDump)
			if len(data) == OPTIONAL_MSG_LENGTH {
				decodeHeatpumpData(optionalPCBTopics, data, mclient)
				//response to heatpump should contain the data from heatpump on byte 4 and 5
				optionalPCBQuery[4] = data[4]
				optionalPCBQuery[5] = data[5]
			} else if len(data) == COMMAND_MSG_LENGTH {
				decodeHeatpumpData(commandTopics, data, mclient)
			}

		}
	}
}

func saveOptionalPCB() {
	err := ioutil.WriteFile(config.optionalPCBFile, optionalPCBQuery, 0644)
	//TODO serialize to json instead, restore topics and []byte
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
