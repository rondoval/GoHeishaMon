package main

import (
	"encoding/binary"
	"fmt"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

var actData [92]string

func clearActData() {
	for {
		time.Sleep(time.Second * time.Duration(config.ForceRefreshTime))
		for k := range actData {
			actData[k] = "nil" //funny i know ;)
		}

	}
}

func callTopicFunction(data byte, f func(data byte) string) string {
	return f(data)
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

func unknown(input byte) string {
	return "-1"
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

func getPumpFlow(data []byte) string { // TOP1 //
	PumpFlow1 := int(data[170])
	PumpFlow2 := ((float64(data[169]) - 1) / 256)
	PumpFlow := float64(PumpFlow1) + PumpFlow2
	return fmt.Sprintf("%.2f", PumpFlow)
}

func getErrorInfo(data []byte) string { // TOP44 //
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

func decodeHeatpumpData(data []byte, mclient mqtt.Client, token mqtt.Token) {

	m := map[string]func(byte) string{
		"getBit7and8":         getBit7and8,
		"unknown":             unknown,
		"getRight3bits":       getRight3bits,
		"getIntMinus1Div5":    getIntMinus1Div5,
		"getIntMinus1Times50": getIntMinus1Times50,
		"getIntMinus1Times10": getIntMinus1Times10,
		"getBit3and4and5":     getBit3and4and5,
		"getIntMinus128":      getIntMinus128,
		"getBit1and2":         getBit1and2,
		"getOpMode":           getOpMode,
		"getIntMinus1":        getIntMinus1,
		"getEnergy":           getEnergy,
		"getBit5and6":         getBit5and6,

		"getBit3and4": getBit3and4,
	}

	for k, v := range allTopics {
		var inputByte byte
		var topicValue string
		var value string
		switch k {
		case 1:
			topicValue = getPumpFlow(data)
		case 11:
			d := make([]byte, 2)
			d[0] = data[183]
			d[1] = data[182]
			topicValue = fmt.Sprintf("%d", int(binary.BigEndian.Uint16(d))-1)
		case 12:
			d := make([]byte, 2)
			d[0] = data[180]
			d[1] = data[179]
			topicValue = fmt.Sprintf("%d", int(binary.BigEndian.Uint16(d))-1)
		case 90:
			d := make([]byte, 2)
			d[0] = data[186]
			d[1] = data[185]
			topicValue = fmt.Sprintf("%d", int(binary.BigEndian.Uint16(d))-1)
		case 91:
			d := make([]byte, 2)
			d[0] = data[189]
			d[1] = data[188]
			topicValue = fmt.Sprintf("%d", int(binary.BigEndian.Uint16(d))-1)
		case 44:
			topicValue = getErrorInfo(data)
		default:
			inputByte = data[v.TopicBit]
			if _, ok := m[v.TopicFunction]; ok {
				topicValue = callTopicFunction(inputByte, m[v.TopicFunction])
			} else {
				fmt.Println("NIE MA FUNKCJI", v.TopicFunction)
			}

		}

		if actData[k] != topicValue {
			actData[k] = topicValue
			fmt.Printf("received TOP%d %s: %s \n", k, v.TopicName, topicValue)
			if config.Aquarea2mqttCompatible {
				TOP := "aquarea/state/" + fmt.Sprintf("%s/%s", config.Aquarea2mqttPumpID, v.TopicA2M)
				value = strings.TrimSpace(topicValue)
				value = strings.ToUpper(topicValue)
				fmt.Println("Publikuje do ", TOP, "warosc", string(value))
				token = mclient.Publish(TOP, byte(0), false, value)
				if token.Wait() && token.Error() != nil {
					fmt.Printf("Fail to publish, %v", token.Error())
				}
			}
			TOP := fmt.Sprintf("%s/%s", config.MqttTopicBase, v.TopicName)
			fmt.Println("Publikuje do ", TOP, "warosc", string(topicValue))
			token = mclient.Publish(TOP, byte(0), false, topicValue)
			if token.Wait() && token.Error() != nil {
				fmt.Printf("Fail to publish, %v", token.Error())
			}

		}

	}

}
