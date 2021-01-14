package main

import (
	"log"
	"time"

	"github.com/tarm/serial"
)

const serialTimeout = 2 * time.Second
const optionalPCBSaveTime = 5 * time.Minute

var panasonicQuery []byte = []byte{0x71, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
var optionalPCBQuery []byte = []byte{0xF1, 0x11, 0x01, 0x50, 0x00, 0x00, 0x40, 0xFF, 0xFF, 0xE5, 0xFF, 0xFF, 0x00, 0xFF, 0xEB, 0xFF, 0xFF, 0x00, 0x00}
var config configStruct

var serialPort *serial.Port
var commandsChannel chan []byte

func main() {
	config = readConfig()
	go updateConfigLoop()

	serialConfig := &serial.Config{Name: config.Device, Baud: 9600, Parity: serial.ParityEven, StopBits: serial.Stop1, ReadTimeout: serialTimeout}
	var err error
	serialPort, err = serial.OpenPort(serialConfig)
	if err != nil {
		log.Println(err)
	}

	commandsChannel = make(chan []byte, 100)
	loadTopics()
	//TODO load PCB data

	mclient := makeMQTTConn()
	if config.HAAutoDiscover == true {
		publishDiscoveryTopics(mclient)
	}

	queryTicker := time.NewTicker(time.Second * time.Duration(config.ReadInterval))
	optionPCBSaveTicker := time.NewTicker(optionalPCBSaveTime)
	for {
		readSerial(mclient)

		var queueLen = len(commandsChannel)
		if queueLen > 50 {
			log.Println("Command queue length: ", len(commandsChannel))
		}

		select {
		case <-optionPCBSaveTicker.C:
			//TODO save PCB data

		case value := <-commandsChannel:
			sendCommand(value)

		case <-queryTicker.C:
			commandsChannel <- panasonicQuery

		default:
			if config.OptionalPCB == true && config.ListenOnly == false {
				commandsChannel <- optionalPCBQuery
			}
		}
		//TODO save optional PCB every 5 minutes
	}
}

// bool saveOptionalPCB(byte* command, int length) {
//   if (LittleFS.begin()) {
//     File pcbfile = LittleFS.open("/optionalpcb.raw", "w");
//     if (pcbfile) {
//       pcbfile.write(command, length);
//       pcbfile.close();
//       return true;
//     }

//   }
//   return false;
// }
// bool loadOptionalPCB(byte* command, int length) {
//   if (LittleFS.begin()) {
//     if (LittleFS.exists("/optionalpcb.raw")) {
//       File pcbfile = LittleFS.open("/optionalpcb.raw", "r");
//       if (pcbfile) {
//         pcbfile.read(command, length);
//         pcbfile.close();
//         return true;
//       }
//     }
//   }
//   return false;
// }
