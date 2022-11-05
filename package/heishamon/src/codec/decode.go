package codec

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"

	"github.com/rondoval/GoHeishaMon/topics"
)

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
	"getBit7and8Z":        func(input byte) int { return int(input & 0b11) },
	"getBit5and6":         func(input byte) int { return int(((input >> 2) & 0b11) - 1) },
	"getBit5and6Z":        func(input byte) int { return int((input >> 2) & 0b11) },
	"getBit3and4":         func(input byte) int { return int(((input >> 4) & 0b11) - 1) },
	"getBit3and4Z":        func(input byte) int { return int((input >> 4) & 0b11) },
	"getBit2and3Z":        func(input byte) int { return int((input >> 5) & 0b11) },
	"getBit1and2":         func(input byte) int { return int((input >> 6) - 1) },
	"getBit1and2Z":        func(input byte) int { return int(input >> 6) },
	"getBit8":             func(input byte) int { return int(input & 0b1) },
	"getBit7":             func(input byte) int { return int((input & 0b10) >> 1) },
	"getBit6":             func(input byte) int { return int((input & 0b100) >> 2) },
	"getBit4":             func(input byte) int { return int((input & 0b1000) >> 4) },
	"getBit2":             func(input byte) int { return int((input & 0b100000) >> 6) },
	"getBit1":             func(input byte) int { return int(input >> 7) },
	"getOpMode":           getOpMode,
	"getModel":            getModel,
	"getPower":            func(input byte) int { return (int(input) - 1) * 200 },
	"hex2temp":            hex2temp,
	"hex2demand":          hex2demand,
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
		return 0 // Heat
	case 19:
		return 1 // Cool
	case 25:
		return 2 // Auto(heat)
	case 33:
		return 3 // DHW
	case 34:
		return 4 // Heat+DHW
	case 35:
		return 5 // Cool+DHW
	case 41:
		return 6 // Auto(heat)+DHW
	case 26:
		return 7 // Auto(cool)
	case 42:
		return 8 // Auto(cool)+DHW
	case 24:
		return 9 // Auto
	case 40:
		return 10 // Auto+DHW
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

func hex2temp(input byte) int {
	const Uref float64 = 255
	const constant float64 = 3695
	const R25 float64 = 6340
	const T25 float64 = 25
	const Rf float64 = 6480
	const K float64 = 273.15
	var hextemp float64 = float64(input)

	var RT float64 = hextemp * Rf / (Uref - hextemp)
	var temp int = int(constant/(math.Log(RT/R25)+constant/(T25+K)) - K)
	return temp
}

func hex2demand(input byte) int {
	var demand int = 0

	const min = 43 - 5 // 0% in hex
	const max = 234    // 100% in hex

	if input >= max {
		demand = 100
	} else if input <= min+5 {
		demand = 5
	} else {
		const a float64 = 95. / (max - min)
		const b float64 = 5 - 95.*min/(max-min)
		demand = int(a*float64(input) + b)
	}

	return demand
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

func convertIntToEnum(value int, topic topics.TopicEntry) string {
	numItems := len(topic.Values())
	if numItems > 0 {
		if value >= 0 && value < numItems {
			return topic.Values()[value]
		}
		log.Printf("Value out of range %s: %d", topic.SensorName(), value)
	}
	return fmt.Sprintf("%d", value)
}

func DecodeHeatpumpData(topics *topics.TopicData, data []byte) *topics.TopicEntry {
	for _, v := range topics.GetAll() {
		var topicValue string

		if v.DecodeFunction() != "" {
			if byteOperator, ok := decodeToInt[v.DecodeFunction()]; ok {
				topicValue = convertIntToEnum(byteOperator(data[v.DecodeOffset()]), *v)
			} else if arrayOperator, ok := decodeToString[v.DecodeFunction()]; ok {
				topicValue = arrayOperator(data, v.DecodeOffset())
			} else {
				log.Print("Unknown codec function: ", v.DecodeFunction())
			}

			if v.CurrentValue != topicValue {
				v.CurrentValue = topicValue
				return v
			}
		}
	}
	return nil
}
