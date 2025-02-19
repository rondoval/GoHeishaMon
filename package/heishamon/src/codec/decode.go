// Package codec implements functions used to decode and encode heat pump data to binary format.
package codec

import (
	"bytes"
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
	"getHiNibble":         func(input byte) int { return int(((input >> 4) & 0b1111) - 1) },
	"getLoNibble":         func(input byte) int { return int((input & 0b1111) - 1) },
	"getOpMode":           getOpMode,
	"getPower":            func(input byte) int { return (int(input) - 1) * 200 },
}

var decodeToFloat = map[string]func(byte) float64{
	"getIntMinus1Div5":  func(input byte) float64 { return float64(input-1) / 5.0 },
	"getIntegral":       func(input byte) float64 { return float64(input) },
	"getFractional":     func(input byte) float64 { return (float64(input) - 1.0) / 256.0 },
	"hex2temp":          hex2temp,
	"hex2demand":        hex2demand,
	"getFractionalLow":  func(input byte) float64 { return getFractional(input & 0b111) },
	"getFractionalHigh": func(input byte) float64 { return getFractional((input >> 3) & 0b111) },
}

var decodeToString = map[string]func([]byte, topics.CodecEntry) string{
	"getWord":      getWord,
	"getErrorInfo": getErrorInfo,
	"getModel":     getModel,
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

func getModel(data []byte, entry topics.CodecEntry) string {
	fingerprint := data[entry.Offset : entry.Offset+10]
	for _, val := range entry.Mapping {
		if bytes.Equal(val.ID, fingerprint) {
			return val.Name
		}
	}
	return fmt.Sprintf("Unknown model with fingerprint: %x", fingerprint)
}

func getFractional(input byte) float64 {
	switch input {
	case 1:
		return 0.0
	case 2:
		return 0.25
	case 3:
		return 0.5
	case 4:
		return 0.75
	}
	return 0
}

func hex2temp(input byte) float64 {
	const Uref float64 = 255
	const constant float64 = 3695
	const R25 float64 = 6340
	const T25 float64 = 25
	const Rf float64 = 6480
	const K float64 = 273.15
	hextemp := float64(input)

	RT := hextemp * Rf / (Uref - hextemp)
	return constant/(math.Log(RT/R25)+constant/(T25+K)) - K
}

func hex2demand(input byte) float64 {
	var demand float64

	const minimum = 43 - 5 // 0% in hex
	const maximum = 234    // 100% in hex

	switch {
	case input >= maximum:
		demand = 100
	case input <= minimum+5:
		demand = 5
	default:
		const a float64 = 95. / (maximum - minimum)
		const b float64 = 5 - 95.*minimum/(maximum-minimum)
		demand = a*float64(input) + b
	}

	return demand
}

func getErrorInfo(data []byte, _ topics.CodecEntry) string {
	errorType := int(data[113])
	errorNumber := int(data[114]) - 17
	var errorString string
	switch errorType {
	case 177: // B1=F type error
		errorString = fmt.Sprintf("F%02X", errorNumber)
	case 161: // A1=H type error
		errorString = fmt.Sprintf("H%02X", errorNumber)
	default:
		errorString = "No error"
	}
	return errorString
}

func getWord(data []byte, entry topics.CodecEntry) string {
	return fmt.Sprintf("%d", int(binary.BigEndian.Uint16([]byte{data[entry.Offset+1], data[entry.Offset]}))-1)
}

func convertIntToEnum(value int, topic *topics.TopicEntry) string {
	numItems := len(topic.Values)
	if numItems > 0 {
		if value >= 0 && value < numItems {
			return topic.Values[value]
		}
		log.Printf("Value out of range %s: %d", topic.SensorName, value)
	}
	return fmt.Sprintf("%d", value)
}

// Decode decodes a byte slice received from the heat pump into a TopicData structure.
// The returned value is a slice containing all TopicEntry records that have changed as a result of decoding.
func Decode(allTopics *topics.TopicData, data []byte, decodeEnums bool) []*topics.TopicEntry {
	changed := make([]*topics.TopicEntry, 0, len(allTopics.GetAll()))
	for _, v := range allTopics.GetAll() {
		if !v.Readable() {
			continue
		}

		var topicValue string
		floatValue := 0.0

		for _, decode := range v.Codec {
			if decode.DecodeFunction == "" {
				continue
			}

			if byteOperator, ok := decodeToInt[decode.DecodeFunction]; ok {
				decoded := byteOperator(data[decode.Offset])
				floatValue += float64(decoded)
				if decodeEnums {
					topicValue = convertIntToEnum(decoded, v)
				} else {
					topicValue = fmt.Sprintf("%d", decoded)
				}
			} else if floatOperator, ok := decodeToFloat[decode.DecodeFunction]; ok {
				floatValue += floatOperator(data[decode.Offset])
				topicValue = fmt.Sprintf("%.2f", floatValue)
			} else if arrayOperator, ok := decodeToString[decode.DecodeFunction]; ok {
				topicValue = arrayOperator(data, decode)
			} else {
				log.Print("Unknown codec function: ", decode.DecodeFunction)
			}
		}

		if v.UpdateValue(topicValue) {
			changed = append(changed, v)
		}
	}
	return changed
}
