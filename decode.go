package main

import (
	"encoding/binary"
	"fmt"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

const numberOfOptionalTopics = 7

var optionalData []string = make([]string, numberOfOptionalTopics)

var funcMapByte = map[string]func(byte) string{
	"getIntMinus1Div5":    getIntMinus1Div5,
	"getIntMinus1Times50": getIntMinus1Times50,
	"getIntMinus1Times10": getIntMinus1Times10,
	"getIntMinus128":      getIntMinus128,
	"getIntMinus1":        getIntMinus1,
	"getOpMode":           getOpMode,
	"getEnergy":           getEnergy,
	"getLeft5bits":        getLeft5bits,
	"getRight3bits":       getRight3bits,
	"getBit3and4and5":     getBit3and4and5,
	"getBit7and8":         getBit7and8,
	"getBit5and6":         getBit5and6,
	"getBit3and4":         getBit3and4,
	"getBit1and2":         getBit1and2,
	"getModel":            getModel,
}

var funcMapArray = map[string]func([]byte, int) string{
	"getPumpFlow":  getPumpFlow,
	"getWord":      getWord,
	"getErrorInfo": getErrorInfo,
}

func getBit1and2(input byte) string {
	return fmt.Sprintf("%d", (input>>6)-1)
}

func getBit3and4(input byte) string {
	return fmt.Sprintf("%d", ((input>>4)&0b11)-1)
}

func getBit5and6(input byte) string {
	return fmt.Sprintf("%d", ((input>>2)&0b11)-1)
}

func getBit7and8(input byte) string {
	return fmt.Sprintf("%d", (input&0b11)-1)
}

func getBit3and4and5(input byte) string {
	return fmt.Sprintf("%d", ((input>>3)&0b111)-1)
}

func getLeft5bits(input byte) string {
	return fmt.Sprintf("%d", (input>>3)-1)
}

func getRight3bits(input byte) string {
	return fmt.Sprintf("%d", (input&0b111)-1)
}

func getIntMinus1(input byte) string {
	value := int(input) - 1
	return fmt.Sprintf("%d", value)
}

func getIntMinus128(input byte) string {
	value := int(input) - 128
	return fmt.Sprintf("%d", value)
}

func getIntMinus1Div5(input byte) string {
	value := int(input) - 1
	var out float32
	out = float32(value) / 5
	return fmt.Sprintf("%.2f", out)
}

func getIntMinus1Times10(input byte) string {
	value := int(input) - 1
	return fmt.Sprintf("%d", value*10)
}

func getIntMinus1Times50(input byte) string {
	value := int(input) - 1
	return fmt.Sprintf("%d", value*50)
}

func getOpMode(input byte) string {
	switch int(input & 0b111111) {
	case 18:
		return "0"
	case 19:
		return "1"
	case 25:
		return "2"
	case 33:
		return "3"
	case 34:
		return "4"
	case 35:
		return "5"
	case 41:
		return "6"
	case 26:
		return "7"
	case 42:
		return "8"
	default:
		return "-1"
	}
}

func getModel(input byte) string {
	switch int(input) {
	case 19:
		return "0"
	case 20:
		return "1"
	case 119:
		return "2"
	case 136:
		return "3"
	case 133:
		return "4"
	case 134:
		return "5"
	case 135:
		return "6"
	case 113:
		return "7"
	case 67:
		return "8"
	case 51:
		return "9"
	case 21:
		return "10"
	case 65:
		return "11"
	case 69:
		return "12"
	case 116:
		return "13"
	case 130:
		return "14"
	default:
		return "-1"
	}
}

func getEnergy(input byte) string {
	value := (int(input) - 1) * 200
	return fmt.Sprintf("%d", value)
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

func decodeHeatpumpData(data []byte, mclient mqtt.Client) {
	for k, v := range allTopics {
		var topicValue string

		if byteOperator, ok := funcMapByte[v.DecodeFunction]; ok {
			topicValue = byteOperator(byte(data[v.DecodeOffset]))
		} else if arrayOperator, ok := funcMapArray[v.DecodeFunction]; ok {
			topicValue = arrayOperator(data, v.DecodeOffset)
		} else {
			log.Println("Unknown codec function: ", v.DecodeFunction)
		}

		if v.currentValue != topicValue {
			v.currentValue = topicValue
			allTopics[k] = v
			topic := fmt.Sprintf("%s/%s", config.mqttValuesTopic, v.SensorName)
			token := mclient.Publish(topic, byte(0), true, topicValue)
			if token.Wait() && token.Error() != nil {
				log.Printf("Fail to publish, %v", token.Error())
			}
		}
	}
}

func decodeOptionalHeatpumpData(data []byte, mclient mqtt.Client) {
	for topicNumber := 0; topicNumber > numberOfOptionalTopics; topicNumber++ {
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
			topic := fmt.Sprintf("%s/%s", config.mqttPcbValuesTopic, name)
			token := mclient.Publish(topic, byte(0), true, value)
			if token.Wait() && token.Error() != nil {
				log.Printf("Fail to publish, %v", token.Error())
			}
		}
	}
	//response to heatpump should contain the data from heatpump on byte 4 and 5
	optionalPCBQuery[4] = data[4]
	optionalPCBQuery[5] = data[5]
}
