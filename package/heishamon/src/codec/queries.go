package codec

import (
	"log"
	"strings"
	"time"

	"github.com/rondoval/GoHeishaMon/logger"
	"github.com/rondoval/GoHeishaMon/mqtt"
	"github.com/rondoval/GoHeishaMon/serial"
	"github.com/rondoval/GoHeishaMon/topics"
)

const (
	optionalDatagramSize  = 19
	panasonicDatagramSize = 110
)

var panasonicQuery = [panasonicDatagramSize]byte{0x71, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
var panasonicEmptyCommand = [panasonicDatagramSize]byte{0xf1, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

type commandHandler struct {
	serialChannel  chan []byte
	commandChannel chan mqtt.Command
	ackChannel     chan []byte

	optionalPCBCommand []byte
	panasonicCommand   []byte

	queryInterval         time.Duration
	optionalQueryInterval time.Duration

	optionalPCB bool

	mqtt mqtt.MQTT
}

// This is used to invoke Start()
type Options struct {
	MQTT          mqtt.MQTT     // This is used to publish entity updates immediately
	QueryInterval time.Duration // Maximum interval between IoT device queries
	AckChannel    chan []byte   // Channel used to receive heat pump data (for acknowledgements on Optional PCB)

	OptionalPCB           bool              // Optional PCB emulation on/off
	OptionalQueryInterval time.Duration     // Optional PCB maximum interval between datagrams
	OptionalTopics        *topics.TopicData // Optional PCB topic data
}

// Start initializes the codec.
// Codec is reposnsible for decoding and encoding datagrams to/from the heat pump.
// It generates heat pump queries and commands at regular intervals - these are sent to the returned channel.
// The ackChannel shall receive all datagrams received from the heatpump targeted at the Optional PCB - this is required to acknowledge heat pump requests.
func Start(options Options) chan []byte {
	c := commandHandler{
		mqtt: options.MQTT,

		commandChannel: options.MQTT.CommandChannel(),
		ackChannel:     options.AckChannel,
		serialChannel:  make(chan []byte, 20),

		queryInterval:    options.QueryInterval,
		panasonicCommand: newPanasonicCommand(),

		optionalPCB:           options.OptionalPCB,
		optionalQueryInterval: options.OptionalQueryInterval,
		optionalPCBCommand:    []byte{0xF1, 0x11, 0x01, 0x50, 0x00, 0x00, 0x40, 0xFF, 0xFF, 0xE5, 0xFF, 0xFF, 0x00, 0xFF, 0xEB, 0xFF, 0xFF, 0x00, 0x00},
	}

	if c.optionalPCB {
		for _, sensor := range options.OptionalTopics.GetAll() {
			if sensor.EncodeFunction != "" && sensor.CurrentValue() != "" {
				encode(sensor, c.optionalPCBCommand)
			}
		}
	}

	go c.encoderThread()
	return c.serialChannel
}

func newPanasonicCommand() []byte {
	cmd := make([]byte, panasonicDatagramSize)
	copy(cmd, panasonicEmptyCommand[:])
	return cmd
}

func (c *commandHandler) encoderThread() {
	c.sendPanasonicQuery()
	if c.optionalPCB {
		c.sendOptionalPCBCommand()
	}

	panasonicTicker := time.NewTicker(time.Second * c.queryInterval)
	optionalTicker := time.NewTicker(time.Second * c.optionalQueryInterval)
	setTimer := time.NewTimer(0)
	if !setTimer.Stop() {
		<-setTimer.C
	}
	var setRunning bool

	for {
		select {
		case command := <-c.commandChannel:
			c.processCommand(command.Topic, command.Payload, command.AllTopics)
			if command.AllTopics.Kind() == topics.Main && !setRunning {
				setTimer.Reset(time.Second * 2)
				setRunning = true
			}
			// TODO speed up optional ticker?

		case <-panasonicTicker.C:
			c.sendPanasonicQuery()

		case <-optionalTicker.C:
			if c.optionalPCB {
				c.sendOptionalPCBCommand()
			}

		case <-setTimer.C:
			panasonicTicker.Reset(time.Second * c.queryInterval)
			setRunning = false
			c.sendPanasonicCommand()

		case datagram := <-c.ackChannel:
			c.acknowledge(datagram)
		}
	}
}

func (c *commandHandler) sendOptionalPCBCommand() {
	toSend := make([]byte, len(c.optionalPCBCommand))
	copy(toSend, c.optionalPCBCommand)
	c.serialChannel <- toSend
}

func (c *commandHandler) sendPanasonicCommand() {
	c.serialChannel <- c.panasonicCommand
	c.panasonicCommand = newPanasonicCommand()
}

func (c *commandHandler) sendPanasonicQuery() {
	c.serialChannel <- panasonicQuery[:]
}

func (c *commandHandler) acknowledge(datagram []byte) {
	// for Optional PCB: response to heatpump should contain the data from heatpump on byte 4 and 5
	if len(datagram) == serial.OptionalMessageLength {
		c.optionalPCBCommand[4] = datagram[4]
		c.optionalPCBCommand[5] = datagram[5]
	} else {
		log.Printf("This does not look like an Optional PCB datagram. Len: %d", len(datagram))
		logger.LogHex("Acknowledge", datagram)
	}
}

func (c *commandHandler) processCommand(mqttTopic, payload string, allTopics *topics.TopicData) {
	topicPieces := strings.Split(mqttTopic, "/")
	sensorName := topicPieces[len(topicPieces)-2]
	log.Printf("Command received - set %s on %s to %s\n", sensorName, allTopics.Kind(), payload)

	sensor, sensorOK := allTopics.Lookup(sensorName)
	if !sensorOK {
		log.Println("Unknown topic: " + sensorName)
		return
	}
	// Update topic data as well for quicker turnaround.
	// Also needed for Optional PCB - the pump does not confirm messages.
	sensor.UpdateValue(payload)
	encode(sensor, c.selectCommand(allTopics.Kind()))
	c.mqtt.PublishValue(sensor)
}

func (c *commandHandler) selectCommand(kind topics.DeviceType) []byte {
	switch kind {
	case topics.Main:
		return c.panasonicCommand

	case topics.Optional:
		return c.optionalPCBCommand
	default:
		return nil
	}
}
