package main

import (
	"encoding/binary"
	"fmt"
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var funcMapByte = map[string]func(byte) string{
	"getIntMinus1Div5":    getIntMinus1Div5,
	"getIntMinus1Times50": getIntMinus1Times50,
	"getIntMinus1Times10": getIntMinus1Times10,
	"getIntMinus128":      getIntMinus128,
	"getIntMinus1":        getIntMinus1,
	"getOpMode":           getOpMode,
	"getEnergy":           getEnergy,
	"getRight3bits":       getRight3bits,
	"getBit3and4and5":     getBit3and4and5,
	"getBit7and8":         getBit7and8,
	"getBit5and6":         getBit5and6,
	"getBit3and4":         getBit3and4,
	"getBit1and2":         getBit1and2,
}

var funcMapArray = map[string]func([]byte, int) string{
	"getPumpFlow":  getPumpFlow,
	"getWord":      getWord,
	"getErrorInfo": getErrorInfo,
	"pumpModel":    pumpModel,
}

func getBit7and8(input byte) string {
	return fmt.Sprintf("%d", (input&0b11)-1)
}

func getBit3and4and5(input byte) string {
	return fmt.Sprintf("%d", ((input>>3)&0b111)-1)
}

func getIntMinus1Times10(input byte) string {
	value := int(input) - 1
	return fmt.Sprintf("%d", value*10)

}

func getIntMinus1Times50(input byte) string {
	value := int(input) - 1
	return fmt.Sprintf("%d", value*50)

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

func getRight3bits(input byte) string {
	return fmt.Sprintf("%d", (input&0b111)-1)

}

func getBit1and2(input byte) string {
	return fmt.Sprintf("%d", (input>>6)-1)

}

func getOpMode(input byte) string {
	switch int(input) {
	case 82:
		return "0"
	case 83:
		return "1"
	case 89:
		return "2"
	case 97:
		return "3"
	case 98:
		return "4"
	case 99:
		return "5"
	case 105:
		return "6"
	case 90:
		return "7"
	case 106:
		return "8"
	default:
		return "-1"
	}
}

func getIntMinus1(input byte) string {
	value := int(input) - 1
	return fmt.Sprintf("%d", value)
}

func getEnergy(input byte) string {
	value := (int(input) - 1) * 200
	return fmt.Sprintf("%d", value)
}

func getBit3and4(input byte) string {
	return fmt.Sprintf("%d", ((input>>4)&0b11)-1)

}

func getBit5and6(input byte) string {
	return fmt.Sprintf("%d", ((input>>2)&0b11)-1)

}

func getWord(data []byte, index int) string {
	return fmt.Sprintf("%d", int(binary.BigEndian.Uint16([]byte{data[index+1], data[index]}))-1)
}

func pumpModel(data []byte, index int) string {
	// TODO
	return "unknown"
}

func getPumpFlow(data []byte, _ int) string { // TOP1 //
	PumpFlow1 := int(data[170])
	PumpFlow2 := ((float64(data[169]) - 1) / 256)
	PumpFlow := float64(PumpFlow1) + PumpFlow2
	return fmt.Sprintf("%.2f", PumpFlow)
}

func getErrorInfo(data []byte, _ int) string { // TOP44 //
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

func decodeHeatpumpData(data []byte, mclient mqtt.Client) {
	for k, v := range allTopics {
		var topicValue string

		if byteOperator, ok := funcMapByte[v.TopicFunction]; ok {
			topicValue = byteOperator(byte(data[v.TopicBit]))
		} else if arrayOperator, ok := funcMapArray[v.TopicFunction]; ok {
			topicValue = arrayOperator(data, v.TopicBit)
		} else {
			log.Println("Unknown codec function: ", v.TopicFunction)
		}

		if v.TopicValue != topicValue {
			v.TopicValue = topicValue
			allTopics[k] = v
			//			log.Printf("received %s: %s \n", v.TopicName, topicValue)
			TOP := fmt.Sprintf("%s/%s", config.MqttTopicBase, v.TopicName)
			token := mclient.Publish(TOP, byte(0), true, topicValue)
			if token.Wait() && token.Error() != nil {
				log.Printf("Fail to publish, %v", token.Error())
			}
		}
	}
}
