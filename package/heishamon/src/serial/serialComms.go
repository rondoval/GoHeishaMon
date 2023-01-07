// Package serial implements serial port communication with the heat pump.
package serial

import (
	"bytes"
	"io"
	"log"
	"time"

	"github.com/rondoval/GoHeishaMon/logger"
	tarm "github.com/tarm/serial"
)

const dataBufferSize = 1024

// OptionalMessageLength is a length of an Optional PCB datagram with checksum
const OptionalMessageLength = 20

// DataMessageLength is a length of an IoT device datagram with a checksum
const DataMessageLength = 203
const loggingRatio = 150

// Comms represents a serial port used to communicate with the heat pump.
// Handles low level communications, i.e. packet assembly, checksum generation/verification etc.
type Comms struct {
	goodreads, totalreads int64
	buffer                bytes.Buffer
	serialPort            *tarm.Port
	serialConfig          *tarm.Config
}

// Open opens the serial port and initializes internal structures.
func (s *Comms) Open(portName string, timeout time.Duration) {
	s.serialConfig = &tarm.Config{Name: portName, Baud: 9600, Parity: tarm.ParityEven, StopBits: tarm.Stop1, ReadTimeout: timeout}
	s.openInternal()
}

func (s *Comms) openInternal() {
	var err error
	s.serialPort, err = tarm.OpenPort(s.serialConfig)
	if err != nil {
		// no point in continuing, no config
		log.Fatal(err)
	}
	log.Print("Serial port open")
	s.serialPort.Flush()
}

// Close closes the serial port.
func (s *Comms) Close() {
	s.serialPort.Close()
}

func isValidReceiveChecksum(data []byte) bool {
	var chk byte
	for _, v := range data {
		chk += v
	}
	return (chk == 0) //all received bytes + checksum should result in 0
}

func calcChecksum(command []byte) byte {
	var chk byte
	for _, v := range command {
		chk += v
	}
	return (chk ^ 0xFF) + 01
}

// SendCommand sends a datagram to the heat pump.
// Appends checksum.
func (s *Comms) SendCommand(command []byte) {
	var chk = calcChecksum(command)

	_, err := s.serialPort.Write(command) //first send command
	if err != nil {
		log.Print(err)
	}
	_, err = s.serialPort.Write([]byte{chk}) //then calculcated checksum byte afterwards
	if err != nil {
		log.Print(err)
	}
	logger.LogHex("Send", command)
}

func (s *Comms) readToBuffer() {
	data := make([]byte, dataBufferSize)
	n, err := s.serialPort.Read(data)
	if err != nil && err != io.EOF {
		log.Print(err)
		s.Close()
		s.openInternal()
	}
	s.buffer.Write(data[:n])
}

func (s *Comms) findHeaderStart() bool {
	if s.buffer.Len() < 1 {
		return false
	}
	hdr := bytes.IndexByte(s.buffer.Bytes(), 0x71)
	if hdr < 0 {
		logger.LogDebug("Header not found in %d bytes", s.buffer.Len())
		return false
	} else if hdr > 0 {
		log.Printf("Throwing away %d bytes of data", hdr)
		waste := s.buffer.Next(hdr)
		logger.LogHex("Waste", waste)
	}
	return true
}

func (s *Comms) dispatchDatagram(len int) []byte {
	s.goodreads++
	readpercentage := float64(s.totalreads-s.goodreads) / float64(s.totalreads) * 100.
	if s.totalreads%loggingRatio == 0 {
		log.Printf("RX: %d RX errors: %d (%.2f %%)", s.totalreads, s.totalreads-s.goodreads, readpercentage)
	}

	packet := s.buffer.Next(len)
	logger.LogHex("Received", packet)
	if len == DataMessageLength || len == OptionalMessageLength {
		logger.LogDebug("Received %d bytes of data with correct header and checksum", len)
		return packet
	}
	log.Printf("Received an unknown datagram. Can't decode this (yet?). Length: %d", len)
	return nil
}

func (s *Comms) checkHeader() (len int, ok bool) {
	// opt header: 71 11 01 50; 20 bytes
	// header:     71 c8 01 10; 203 bytes
	data := s.buffer.Bytes()
	len = int(data[1]) + 3
	ok = false
	if data[0] == 0x71 && data[2] == 0x1 && (data[3] == 0x50 || data[3] == 0x10) {
		ok = true
		return
	}
	logger.LogDebug("Bad header: %x", data[:4])
	return
}

// Read attempts to read heat pump reply. Returns nil if full packet with correct checksum was not assembled.
// It holds state and should be called periodically.
func (s *Comms) Read(logHexDump bool) []byte {
	s.readToBuffer()

	if s.findHeaderStart() && s.buffer.Len() >= 4 { // have entire header at start of buffer
		var (
			len int
			ok  bool
		)

		if len, ok = s.checkHeader(); !ok {
			//consume byte, it's not a header
			s.buffer.ReadByte()
			return nil
		}

		if s.buffer.Len() >= len { // have entire packet
			s.totalreads++

			if isValidReceiveChecksum(s.buffer.Bytes()[:len]) {
				return s.dispatchDatagram(len)
			}
			// invalid checksum, need to consume 0x71 and look for another one
			s.buffer.ReadByte()
			log.Println("Invalid checksum on receive!")

		} else {
			logger.LogDebug("Awaiting full packet. Have %d, missing %d", s.buffer.Len(), len-s.buffer.Len())
		}
	}
	return nil
}
