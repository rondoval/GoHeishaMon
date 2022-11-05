package serial

import (
	"bytes"
	"io"
	"log"
	"os/exec"
	"time"

	"github.com/rondoval/GoHeishaMon/logger"
	tarm "github.com/tarm/serial"
)

const DATA_BUFFER = 1024
const OPTIONAL_MSG_LENGTH = 20
const DATA_MSG_LENGTH = 203
const LOGGING_RATIO = 150

type SerialComms struct {
	goodreads, totalreads int64
	buffer                bytes.Buffer
	serialPort            *tarm.Port
}

func (s *SerialComms) Open(portName string, timeout time.Duration) {
	serialConfig := &tarm.Config{Name: portName, Baud: 9600, Parity: tarm.ParityEven, StopBits: tarm.Stop1, ReadTimeout: timeout}
	var err error
	s.serialPort, err = tarm.OpenPort(serialConfig)
	if err != nil {
		// no point in continuing, no config
		log.Fatal(err)
	}
	log.Print("Serial port open")
}

func (s *SerialComms) Close() {
	s.serialPort.Close()
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

func (s *SerialComms) SendCommand(command []byte) {
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

func (s *SerialComms) readToBuffer() {
	data := make([]byte, DATA_BUFFER)
	n, err := s.serialPort.Read(data)
	if err != nil && err != io.EOF {
		log.Print(err)
		time.Sleep(10 * time.Second)
		exec.Command("reboot").Run()
		// TODO replace with some sort of receovery - reopen serial port?
	}
	s.buffer.Write(data[:n])
}

func (s *SerialComms) findHeaderStart() bool {
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

func (s *SerialComms) dispatchDatagram(len int) []byte {
	s.goodreads++
	readpercentage := float64(s.totalreads-s.goodreads) / float64(s.totalreads) * 100.
	if s.totalreads%LOGGING_RATIO == 0 {
		log.Printf("RX: %d RX errors: %d (%.2f %%)", s.totalreads, s.totalreads-s.goodreads, readpercentage)
	}

	packet := s.buffer.Next(len)
	logger.LogHex("Received", packet)
	if len == DATA_MSG_LENGTH || len == OPTIONAL_MSG_LENGTH {
		logger.LogDebug("Received %d bytes of data with correct header and checksum", len)
		return packet
	} else {
		log.Printf("Received an unknown datagram. Can't decode this (yet?). Length: %d", len)
		return nil
	}
}

func (s *SerialComms) checkHeader() (len int, ok bool) {
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

func (s *SerialComms) Read(logHexDump bool) []byte {
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
			} else {
				// invalid checksum, need to consume 0x71 and look for another one
				s.buffer.ReadByte()
				log.Println("Invalid checksum on receive!")
			}
		} else {
			//TODO			logger.LogDebug("Awaiting full packet. Have %d, missing %d", s.buffer.Len(), len-s.buffer.Len())
		}
	}
	return nil
}
