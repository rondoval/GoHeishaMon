package main

import (
	"fmt"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var goodreads float64
var totalreads float64
var readpercentage float64

func logHex(command []byte) {
	if config.Loghex {
		log.Printf("%X\n", command)
	}
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

func calcChecksum(command []byte) byte {
	var chk byte = 0
	for _, v := range command {
		chk += v
	}
	return (chk ^ 0xFF) + 01
}

func sendCommand(command []byte) {
	var chk = calcChecksum(command)

	_, err := serialPort.Write(command) //first send command
	if err != nil {
		log.Println(err)
	}
	_, err = serialPort.Write([]byte{chk}) //then calculcated checksum byte afterwards
	if err != nil {
		log.Println(err)
	}
	logHex(command)
}

func readSerial(mclient mqtt.Client) bool {
	const dataLength = 203

	totalreads++
	data := make([]byte, dataLength)
	n, err := serialPort.Read(data)
	if err != nil {
		//TODO reopen port?
		log.Fatal(err)
	}
	if n != dataLength {
		// no data received or synchornizing with the stream
		return false
	}

	logHex(data)
	if !isValidReceiveHeader(data) {
		log.Println("Received wrong header!")
		return false
	}
	if !isValidReceiveChecksum(data) {
		log.Println("Checksum received false!")
		return false
	}
	log.Println("Checksum and header received ok!")
	goodreads++
	readpercentage = ((goodreads / totalreads) * 100)
	log.Println(fmt.Sprintf("Total reads : %f and total good reads : %f (%.2f %%)", totalreads, goodreads, readpercentage))
	decodeHeatpumpData(data, mclient)
	return true
}
