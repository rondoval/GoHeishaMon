// Package codec implements functions used to decode and encode heat pump data to binary format.
package codec

import (
	"errors"
	"log"
	"math"
	"strconv"

	"github.com/rondoval/GoHeishaMon/topics"
)

var encodeInt = map[string]func(int, byte) byte{
	"setIntDiv50Plus1": func(input int, _ byte) byte { return byte(input/50) + 1 },
	"setIntDiv30Plus1": func(input int, _ byte) byte { return byte(input/30) + 1 },
	"setIntDiv10Plus1": func(input int, _ byte) byte { return byte(input/10) + 1 },
	"setIntPlus128":    func(input int, _ byte) byte { return byte(input) + 128 },
	"setIntPlus1":      func(input int, _ byte) byte { return byte(input) + 1 },
	"setRight3bits":    func(input int, val byte) byte { return val&0xf8 | byte(input+1)&7 },
	"setLeft5bits":     func(input int, val byte) byte { return val&0x7 | byte(input+1)&31<<3 },
	"setBit3and4and5":  func(input int, val byte) byte { return val&0xc7 | byte(input+1)&7<<3 },
	"setBit7and8":      func(input int, val byte) byte { return val&0xfc | byte(input+1)&3 },
	"setBit7and8Z":     func(input int, val byte) byte { return val&0xfc | byte(input)&3 },
	"setBit5and6":      func(input int, val byte) byte { return val&0xf3 | byte(input+1)&3<<2 },
	"setBit5and6Z":     func(input int, val byte) byte { return val&0xf3 | byte(input)&3<<2 },
	"setBit3and4":      func(input int, val byte) byte { return val&0xcf | byte(input+1)&3<<4 },
	"setBit3and4Z":     func(input int, val byte) byte { return val&0xcf | byte(input)&3<<4 },
	"setBit2and3Z":     func(input int, val byte) byte { return val&0x9f | byte(input)&3<<5 },
	"setBit1and2":      func(input int, val byte) byte { return val&0x3f | byte(input+1)&3<<6 },
	"setBit1and2Z":     func(input int, val byte) byte { return val&0x3f | byte(input)&3<<6 },
	"setBit8":          func(input int, val byte) byte { return val&0xfe | byte(input)&1 },
	"setBit7":          func(input int, val byte) byte { return val&0xfd | byte(input)&1<<1 },
	"setBit6":          func(input int, val byte) byte { return val&0xfb | byte(input)&1<<2 },
	"setBit5":          func(input int, val byte) byte { return val&0xf7 | byte(input)&1<<3 },
	"setBit4":          func(input int, val byte) byte { return val&0xef | byte(input)&1<<4 },
	"setBit2":          func(input int, val byte) byte { return val&0xbf | byte(input)&1<<6 },
	"setBit1":          func(input int, val byte) byte { return val&0x7f | byte(input)&1<<7 },
	"setHiNibble":      func(input int, val byte) byte { return val&0xf | byte(input+1)&0xf<<4 },
	"setLoNibble":      func(input int, val byte) byte { return val&0xf0 | byte(input+1)&0xf },
	"setOpMode":        setOperationMode,
}

var encodeFloat = map[string]func(float64) byte{
	"temp2hex":   temp2hex,
	"demand2hex": demand2hex,
}

func temp2hex(temp float64) byte {
	var hextemp byte

	switch {
	case temp >= 120:
		hextemp = 0
	case temp <= -78:
		hextemp = 255
	default:
		const Uref float64 = 255
		const constant float64 = 3695
		const R25 float64 = 6340
		const T25 float64 = 25
		const Rf float64 = 6480
		const K float64 = 273.15
		RT := R25 * math.Exp(constant*(1/(temp+K)-1/(T25+K)))
		hextemp = byte(Uref * (RT / (Rf + RT)))
	}

	return hextemp
}

func demand2hex(demand float64) byte {
	var hexdemand byte

	const minimum = 43 - 5 // 0% in hex
	const maximum = 234    // 100% in hex

	switch {
	case demand >= 100:
		hexdemand = maximum
	case demand <= 5:
		hexdemand = minimum + 5
	default:
		const a float64 = (maximum - minimum) / 100.
		hexdemand = byte(a*demand + minimum)
	}

	return hexdemand
}

func setOperationMode(val int, _ byte) (data byte) {
	switch val {
	case 0:
		data = 18 // Heat
	case 1:
		data = 19 // Cool
	case 2:
		data = 24 // Auto(heat) -> Auto
	case 3:
		data = 33 // DHW
	case 4:
		data = 34 // Heat+DHW
	case 5:
		data = 35 // Cool+DHW
	case 6:
		data = 40 // Auto(heat)+DHW -> Auto+DHW
	case 7:
		data = 24 // Auto(cool) -> Auto
	case 8:
		data = 40 // Auto(cool)+DHW -> Auto+DHW
	case 9:
		data = 24 // Auto
	case 10:
		data = 40 // Auto+DHW
	default:
		data = 0
	}
	return data
}

func verboseToNumber(value string, sensor *topics.TopicEntry) (int, error) {
	if number, err := strconv.ParseInt(value, 10, 16); err == nil {
		return int(number), nil
	}

	for valueKey, valueName := range sensor.Values {
		if value == valueName {
			return valueKey, nil
		}
	}
	return 0, errors.New("can't convert literal to number")
}

func encode(sensor *topics.TopicEntry, command []byte) {
	if !sensor.Writable() {
		log.Println("No encode function specified: " + sensor.SensorName)
		return
	}
	msg := sensor.CurrentValue()

	for _, codec := range sensor.Codec {
		if handler, ok := encodeInt[codec.EncodeFunction]; ok {
			v, err := verboseToNumber(msg, sensor)
			if err != nil {
				log.Println(err)
				return
			}
			data := handler(v, command[codec.Offset])
			log.Printf("Setting offset %d to %d", codec.Offset, data)
			command[codec.Offset] = data
			return
		} else if handler, ok := encodeFloat[codec.EncodeFunction]; ok {
			v, err := strconv.ParseFloat(msg, 64)
			if err != nil {
				log.Println(err)
				return
			}
			data := handler(v)
			log.Printf("Setting offset %d to %d", codec.Offset, data)
			command[codec.Offset] = data
			return
		}

		log.Println("No encoder implemented for " + codec.EncodeFunction)
	}
}
