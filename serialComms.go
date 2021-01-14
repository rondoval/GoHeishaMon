package main

import (
	"fmt"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var goodreads, totalreads, readpercentage float64

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
	goodreads++
	readpercentage = ((goodreads / totalreads) * 100)
	//TODO stats over mqtt
	log.Println(fmt.Sprintf("Total reads : %f and total good reads : %f (%.2f %%)", totalreads, goodreads, readpercentage))
	decodeHeatpumpData(data, mclient)
	return true
}

func readSerialNew(mclient mqtt.Client) {
	const maxDataLength = 255

	totalreads++
	data := make([]byte, maxDataLength)
	n, err := serialPort.Read(data)
	if err != nil {
		//TODO reopen port?
		log.Fatal(err)
	}
	if n == 0 {
		//no data
		return
	}

	if data[0] != 113 { //wrong header received!
		log.Println("Received bad header. Ignoring this data!")
		logHex(data[:n])
		return
	}

	if n > 1 { //should have received length part of header now

		if (n > int(data[1]+3)) || (n >= maxDataLength) {
			log.Println("Received more data than header suggests! Ignoring this as this is bad data.")
			logHex(data[:n])
			return
		}

		if n == int(data[1]+3) { //we received all data (data[1] is header length field)
			log.Printf("Received %d bytes data", n)
			logHex(data[:n])
			if !isValidReceiveChecksum(data[:n]) {
				log.Println("Checksum received false!")
				return
			}
			log.Println("Checksum and header received ok!")
			goodreads++
			readpercentage = goodreads / totalreads * 100
			log.Println(fmt.Sprintf("Total reads : %f and total good reads : %f (%.2f %%)", totalreads, goodreads, readpercentage))

			if n == 203 { //for now only return true for this datagram because we can not decode the shorter datagram yet
				decodeHeatpumpData(data[:n], mclient)
			} else if n == 20 { //optional pcb acknowledge answer
				log.Println("Received optional PCB ack answer. Decoding this in OPT topics.")
				decodeOptionalHeatpumpData(data[:n], mclient)
			} else {
				log.Println("Received a shorter datagram. Can't decode this yet.")
			}
		}
	}
}
