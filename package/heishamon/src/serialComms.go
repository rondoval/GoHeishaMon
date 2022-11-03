package main

import (
	"io"
	"log"
	"os/exec"
	"time"
)

var goodreads, totalreads int64

const OPTIONAL_MSG_LENGTH = 20
const COMMAND_MSG_LENGTH = 203
const LOGGING_RATIO = 150

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

func readSerial(logHexDump bool) []byte {
	const maxDataLength = 255

	data := make([]byte, maxDataLength)
	n, err := serialPort.Read(data)
	if err != nil {
		if err != io.EOF {
			log.Print(err)
			time.Sleep(5 * time.Second)
			exec.Command("reboot").Run()
		}
		//TODO rework to detect header and read len of bytes from that, remove sleeps
		return nil
	}
	if n == 0 {
		//no data
		return nil
	}
	totalreads++

	if data[0] != 113 { //wrong header received!
		log.Print("Received bad header. Ignoring this data!")
		logHex(data[:n])
		return nil
	}

	if n > 1 { //should have received length part of header now

		if (n > int(data[1]+3)) || (n >= maxDataLength) {
			log.Print("Received more data than header suggests! Ignoring this as this is bad data.")
			logHex(data[:n])
			return nil
		}

		if n == int(data[1]+3) { //we received all data (data[1] is header length field)
			if logHexDump == true {
				log.Printf("Received %d bytes data", n)
				logHex(data[:n])
			}
			if !isValidReceiveChecksum(data[:n]) {
				log.Println("Invalid checksum on receive!")
				return nil
			}
			goodreads++
			readpercentage := float64(totalreads-goodreads) / float64(totalreads) * 100.
			if totalreads%LOGGING_RATIO == 0 {
				log.Printf("RX: %d RX errors: %d (%.2f %%)", totalreads, totalreads-goodreads, readpercentage)
			}

			if n == COMMAND_MSG_LENGTH || n == OPTIONAL_MSG_LENGTH {
				return data[:n]
			} else {
				log.Print("Received a shorter datagram. Can't decode this yet.")
			}
		}
	}
	return nil
}
