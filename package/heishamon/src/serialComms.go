package main

import (
	"bytes"
	"io"
	"log"
	"os/exec"
	"time"
)

var goodreads, totalreads int64
var buffer bytes.Buffer

const DATA_BUFFER = 512
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

func readToBuffer() {
	data := make([]byte, DATA_BUFFER)
	n, err := serialPort.Read(data)
	if err != nil && err != io.EOF {
		log.Print(err)
		time.Sleep(5 * time.Second)
		exec.Command("reboot").Run()
	}
	buffer.Write(data[:n])
}

func readSerial(logHexDump bool) []byte {
	readToBuffer()
	if buffer.Len() < OPTIONAL_MSG_LENGTH {
		// not enough data to have a full message
		return nil
	}

	// opt header: 71 11 01 50; 20 bytes
	// header:     71 c8 01 10; 203 bytes
	// looking for header
	hdr := bytes.IndexByte(buffer.Bytes(), 0x71)
	if hdr < 0 {
		if config.LogDebug {
			log.Println("Header not found")
		}
		return nil
	} else if hdr > 0 {
		log.Print("Throwing away some data")
		logHex(buffer.Next(hdr))
	}

	if buffer.Len() > 1 { // can check length
		data := buffer.Bytes()
		lenFromHeader := data[1] + 3
		if config.LogDebug {
			log.Printf("Date len in header: %d", lenFromHeader)
		}

		if buffer.Len() >= int(lenFromHeader) { // have entire packet
			totalreads++

			if isValidReceiveChecksum(data[:lenFromHeader]) {
				goodreads++
				readpercentage := float64(totalreads-goodreads) / float64(totalreads) * 100.
				if totalreads%LOGGING_RATIO == 0 {
					log.Printf("RX: %d RX errors: %d (%.2f %%)", totalreads, totalreads-goodreads, readpercentage)
				}

				packet := buffer.Next(int(lenFromHeader))
				if logHexDump == true {
					log.Printf("Received %d bytes data", lenFromHeader)
					logHex(packet)
				}
				if lenFromHeader == COMMAND_MSG_LENGTH || lenFromHeader == OPTIONAL_MSG_LENGTH {
					return packet
				} else {
					log.Print("Received an unknown datagram. Can't decode this (yet?).")
				}
			} else {
				// invalid checksum, need to consume 0x71 and look for another one
				buffer.ReadByte()
				log.Println("Invalid checksum on receive!")
				return nil
			}
		}
	}
	return nil
}
