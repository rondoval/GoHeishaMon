package codec

import (
	"log"
	"strings"
	"time"

	"github.com/rondoval/GoHeishaMon/mqtt"
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

	mqtt mqtt.MQTT
}

func Start(mqtt mqtt.MQTT, queryInterval, optionalQueryInterval time.Duration, optionalTopics *topics.TopicData, ackChannel chan []byte) chan []byte {
	var c commandHandler
	c.mqtt = mqtt
	c.commandChannel = mqtt.CommandChannel()
	c.ackChannel = ackChannel
	c.serialChannel = make(chan []byte, 20)

	c.optionalQueryInterval = optionalQueryInterval
	c.queryInterval = queryInterval

	c.optionalPCBCommand = []byte{0xF1, 0x11, 0x01, 0x50, 0x00, 0x00, 0x40, 0xFF, 0xFF, 0xE5, 0xFF, 0xFF, 0x00, 0xFF, 0xEB, 0xFF, 0xFF, 0x00, 0x00}
	for _, sensor := range optionalTopics.GetAll() {
		if sensor.EncodeFunction != "" && sensor.CurrentValue() != "" {
			encode(sensor, c.optionalPCBCommand)
		}
	}

	c.panasonicCommand = newPanasonicCommand()

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
	c.sendOptionalPCBCommand()

	panasonicTicker := time.NewTicker(time.Second * c.queryInterval)
	optionalTicker := time.NewTicker(time.Second * c.optionalQueryInterval)
	var setTimer *time.Timer

	for {
		select {
		case command := <-c.commandChannel:
			c.processCommand(command.Topic, command.Payload, command.AllTopics)
			if command.AllTopics.Kind() == topics.Main {
				if setTimer != nil && !setTimer.Stop() {
					<-setTimer.C
				}
				setTimer = time.NewTimer(time.Second * 2)
			}
			// TODO speed up optional ticker?

		case <-panasonicTicker.C:
			c.sendPanasonicQuery()

		case <-optionalTicker.C:
			c.sendOptionalPCBCommand()

		case <-setTimer.C:
			panasonicTicker.Reset(time.Second * c.queryInterval)
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
	//response to heatpump should contain the data from heatpump on byte 4 and 5
	c.optionalPCBCommand[4] = datagram[4]
	c.optionalPCBCommand[5] = datagram[5]
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
