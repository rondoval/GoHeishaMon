package main

import (
	"io"
	"log"
	"os/exec"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var goodreads, totalreads int64

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
		log.Print(err)
	}
	_, err = serialPort.Write([]byte{chk}) //then calculcated checksum byte afterwards
	if err != nil {
		log.Print(err)
	}
	logHex(command)
}

func readSerial(mclient mqtt.Client) {
	const maxDataLength = 255

	data := make([]byte, maxDataLength)
	n, err := serialPort.Read(data)
	if err != nil {
		if err != io.EOF {
			log.Print(err)
			time.Sleep(5 * time.Second)
			exec.Command("reboot").Run()
		}
		return
	}
	if n == 0 {
		//no data
		return
	}
	totalreads++

	if data[0] != 113 { //wrong header received!
		log.Print("Received bad header. Ignoring this data!")
		logHex(data[:n])
		return
	}

	if n > 1 { //should have received length part of header now

		if (n > int(data[1]+3)) || (n >= maxDataLength) {
			log.Print("Received more data than header suggests! Ignoring this as this is bad data.")
			logHex(data[:n])
			return
		}

		if n == int(data[1]+3) { //we received all data (data[1] is header length field)
			if config.LogHexDump == true {
				log.Printf("Received %d bytes data", n)
				logHex(data[:n])
			}
			if !isValidReceiveChecksum(data[:n]) {
				log.Println("Invalid checksum on receive!")
				return
			}
			goodreads++
			readpercentage := float64(totalreads-goodreads) / float64(totalreads) * 100.
			log.Printf("RX: %d RX errors: %d (%.2f %%)", totalreads, totalreads-goodreads, readpercentage)

			if n == 203 { //for now only return true for this datagram because we can not decode the shorter datagram yet
				decodeHeatpumpData(data[:n], mclient)
			} else if n == 20 { //optional pcb acknowledge answer
				log.Print("Received optional PCB ack answer. Decoding this in OPT topics.")
				decodeOptionalHeatpumpData(data[:n], mclient)
			} else {
				log.Print("Received a shorter datagram. Can't decode this yet.")
			}
		}
	}
}
