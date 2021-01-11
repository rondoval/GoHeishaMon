package main

import (
	"fmt"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var goodreads float64
var totalreads float64
var readpercentage float64

func logMessage(a string) {
	fmt.Println(a)
}

func logHex(command []byte, length int) {
	fmt.Printf("% X \n", string(command))

}

func calcChecksum(command []byte) byte {
	var chk byte = 0
	for _, v := range command {
		chk += v
	}
	return (chk ^ 0xFF) + 01
}

func sendCommand(command []byte, length int) bool {

	var chk byte
	chk = calcChecksum(command)
	var bytesSent int

	bytesSent, err := serialPort.Write(command) //first send command
	_, err = serialPort.Write([]byte{chk})      //then calculcated checksum byte afterwards
	if err != nil {
		fmt.Println(err)
	}
	logMsg := fmt.Sprintf("sent bytes: %d with checksum: %d ", bytesSent, int(chk))
	logMessage(logMsg)

	if config.Loghex == true {
		logHex(command, length)
	}
	return true
}

func readSerial(MC mqtt.Client, MT mqtt.Token) bool {

	dataLength := 203

	totalreads++
	data := make([]byte, dataLength)
	n, err := serialPort.Read(data)
	if err != nil {
		log.Fatal(err)
	}
	if n == 0 {
		fmt.Println("\nEOF")

	}

	//panasonic read is always 203 on valid receive, if not yet there wait for next read
	logMessage("Received 203 bytes data\n")
	if config.Loghex {
		logHex(data, dataLength)
	}
	if !isValidReceiveHeader(data) {
		logMessage("Received wrong header!\n")
		dataLength = 0 //for next attempt;
		return false
	}
	if !isValidReceiveChecksum(data) {
		logMessage("Checksum received false!")
		dataLength = 0 //for next attempt
		return false
	}
	logMessage("Checksum and header received ok!")
	dataLength = 0 //for next attempt
	goodreads++
	readpercentage = ((goodreads / totalreads) * 100)
	logMsg := fmt.Sprintf("Total reads : %f and total good reads : %f (%.2f %%)", totalreads, goodreads, readpercentage)
	logMessage(logMsg)
	decodeHeatpumpData(data, MC, MT)
	token := MC.Publish(fmt.Sprintf("%s/LWT", config.MqttSetBase), byte(0), false, "Online")
	if token.Wait() && token.Error() != nil {
		fmt.Printf("Fail to publish, %v", token.Error())
	}
	return true

}

func isValidReceiveHeader(data []byte) bool {
	return ((data[0] == 0x71) && (data[1] == 0xC8) && (data[2] == 0x01) && (data[3] == 0x10))
}

func isValidReceiveChecksum(data []byte) bool {
	var chk byte = 0
	for _, v := range data {
		chk += v
	}
	return (chk == 0) //all received bytes + checksum should result in 0
}
