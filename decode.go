package main

import (
	"encoding/binary"
	"fmt"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const numberOfOptionalTopics = 7

var optionalData []string = make([]string, numberOfOptionalTopics)

var decodeToInt = map[string]func(byte) int{
	"getIntMinus1Times50": func(input byte) int { return (int(input) - 1) * 50 },
	"getIntMinus1Times30": func(input byte) int { return (int(input) - 1) * 30 },
	"getIntMinus1Times10": func(input byte) int { return (int(input) - 1) * 10 },
	"getIntMinus128":      func(input byte) int { return int(input) - 128 },
	"getIntMinus1":        func(input byte) int { return int(input) - 1 },
	"getRight3bits":       func(input byte) int { return int((input & 0b111) - 1) },
	"getLeft5bits":        func(input byte) int { return int((input >> 3) - 1) },
	"getBit3and4and5":     func(input byte) int { return int(((input >> 3) & 0b111) - 1) },
	"getBit7and8":         func(input byte) int { return int((input & 0b11) - 1) },
	"getBit5and6":         func(input byte) int { return int(((input >> 2) & 0b11) - 1) },
	"getBit3and4":         func(input byte) int { return int(((input >> 4) & 0b11) - 1) },
	"getBit1and2":         func(input byte) int { return int((input >> 6) - 1) },
	"getBit7":             func(input byte) int { return int((input & 0b10) >> 1) },
	"getBit6":             func(input byte) int { return int((input & 0b100) >> 2) },
	"getOpMode":           getOpMode,
	"getModel":            getModel,
	"getPower":            func(input byte) int { return (int(input) - 1) * 200 },
}

var decodeToString = map[string]func([]byte, int) string{
	"getIntMinus1Div5": getIntMinus1Div5,
	"getPumpFlow":      getPumpFlow,
	"getWord":          getWord,
	"getErrorInfo":     getErrorInfo,
}

func getOpMode(input byte) int {
	switch int(input & 0b111111) {
	case 18:
		return 0
	case 19:
		return 1
	case 25:
		return 2
	case 33:
		return 3
	case 34:
		return 4
	case 35:
		return 5
	case 41:
		return 6
	case 26:
		return 7
	case 42:
		return 8
	default:
		return -1
	}
}

func getModel(input byte) int {
	switch int(input) {
	case 19:
		return 0
	case 20:
		return 1
	case 119:
		return 2
	case 136:
		return 3
	case 133:
		return 4
	case 134:
		return 5
	case 135:
		return 6
	case 113:
		return 7
	case 67:
		return 8
	case 51:
		return 9
	case 21:
		return 10
	case 65:
		return 11
	case 69:
		return 12
	case 116:
		return 13
	case 130:
		return 14
	default:
		return -1
	}
}

func getIntMinus1Div5(data []byte, index int) string {
	value := int(data[index]) - 1
	return fmt.Sprintf("%.2f", float32(value)/5)
}

func getPumpFlow(data []byte, _ int) string {
	PumpFlow1 := int(data[170])
	PumpFlow2 := ((float64(data[169]) - 1) / 256)
	PumpFlow := float64(PumpFlow1) + PumpFlow2
	return fmt.Sprintf("%.2f", PumpFlow)
}

func getErrorInfo(data []byte, _ int) string {
	errorType := int(data[113])
	errorNumber := int(data[114]) - 17
	var errorString string
	switch errorType {
	case 177: //B1=F type error
		errorString = fmt.Sprintf("F%02X", errorNumber)

	case 161: //A1=H type error
		errorString = fmt.Sprintf("H%02X", errorNumber)

	default:
		errorString = fmt.Sprintf("No error")

	}
	return errorString
}

func getWord(data []byte, index int) string {
	return fmt.Sprintf("%d", int(binary.BigEndian.Uint16([]byte{data[index+1], data[index]}))-1)
}

func convertIntToEnum(value int, topic topicData) string {
	numItems := len(topic.Values)
	if numItems > 0 {
		if value >= 0 && value < numItems {
			return topic.Values[value]
		}
		log.Printf("Value out of range %s: %d\n", topic.SensorName, value)
	}
	return fmt.Sprintf("%d", value)
}

func decodeHeatpumpData(data []byte, mclient mqtt.Client) {
	for k, v := range allTopics {
		var topicValue string

		if byteOperator, ok := decodeToInt[v.DecodeFunction]; ok {
			topicValue = convertIntToEnum(byteOperator(data[v.DecodeOffset]), v)
		} else if arrayOperator, ok := decodeToString[v.DecodeFunction]; ok {
			topicValue = arrayOperator(data, v.DecodeOffset)
		} else {
			log.Println("Unknown codec function: ", v.DecodeFunction)
		}

		if v.currentValue != topicValue {
			v.currentValue = topicValue
			allTopics[k] = v
			mqttPublish(mclient, getStatusTopic(v.SensorName), topicValue, 0)
		}
	}
}

func decodeOptionalHeatpumpData(data []byte, mclient mqtt.Client) {
	for topicNumber := 0; topicNumber < numberOfOptionalTopics; topicNumber++ {
		var value, name string

		switch topicNumber {
		case 0:
			value = fmt.Sprintf("%d", data[4]>>7)
			name = "Z1_Water_Pump"
		case 1:
			value = fmt.Sprintf("%d", (data[4]>>5)&0b11)
			name = "Z1_Mixing_Valve"
		case 2:
			value = fmt.Sprintf("%d", (data[4]>>4)&0b1)
			name = "Z2_Water_Pump"
		case 3:
			value = fmt.Sprintf("%d", (data[4]>>2)&0b11)
			name = "Z2_Mixing_Valve"
		case 4:
			value = fmt.Sprintf("%d", (data[4]>>1)&0b1)
			name = "Pool_Water_Pump"
		case 5:
			value = fmt.Sprintf("%d", (data[4]>>0)&0b1)
			name = "Solar_Water_Pump"
		case 6:
			value = fmt.Sprintf("%d", (data[5]>>0)&0b1)
			name = "Alarm_State"
		}

		if optionalData[topicNumber] != value {
			optionalData[topicNumber] = value
			mqttPublish(mclient, getPcbStatusTopic(name), value, 0)
		}
	}
	//response to heatpump should contain the data from heatpump on byte 4 and 5
	optionalPCBQuery[4] = data[4]
	optionalPCBQuery[5] = data[5]
}
