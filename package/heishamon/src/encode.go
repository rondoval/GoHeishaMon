package main

import (
	"math"
)

var encodeInt = map[string]func(int, byte) byte{
	"setIntDiv50Plus1": func(input int, _ byte) byte { return byte(input/50) + 1 },
	"setIntDiv30Plus1": func(input int, _ byte) byte { return byte(input/30) + 1 },
	"setIntDiv10Plus1": func(input int, _ byte) byte { return byte(input/10) + 1 },
	"setIntPlus128":    func(input int, _ byte) byte { return byte(input) + 128 },
	"setIntPlus1":      func(input int, _ byte) byte { return byte(input) + 1 },
	"setRight3bits":    func(input int, val byte) byte { return val&0xf8 | byte(input+1)&0x7 },
	"setLeft5bits":     func(input int, val byte) byte { return val&0x7 | byte(input+1)&0xf8 },
	"setBit3and4and5":  func(input int, val byte) byte { return val&0xc7 | byte(input+1)&0x38 },
	"setBit7and8":      func(input int, val byte) byte { return val&0xfc | byte(input+1)&0x3 },
	"setBit7and8Z":     func(input int, val byte) byte { return val&0xfc | byte(input)&0x3 },
	"setBit5and6":      func(input int, val byte) byte { return val&0xf3 | byte(input+1)&0xc },
	"setBit5and6Z":     func(input int, val byte) byte { return val&0xf3 | byte(input)&0xc },
	"setBit3and4":      func(input int, val byte) byte { return val&0xcf | byte(input+1)&0x30 },
	"setBit3and4Z":     func(input int, val byte) byte { return val&0xcf | byte(input)&0x30 },
	"setBit2and3Z":     func(input int, val byte) byte { return val&0x9f | byte(input)&0x60 },
	"setBit1and2":      func(input int, val byte) byte { return val&0x3f | byte(input+1)&0xc0 },
	"setBit1and2Z":     func(input int, val byte) byte { return val&0x3f | byte(input)&0xc0 },
	"setBit8":          func(input int, val byte) byte { return val&0xfe | byte(input)&0x1 },
	"setBit7":          func(input int, val byte) byte { return val&0xfd | byte(input)&0x2 },
	"setBit6":          func(input int, val byte) byte { return val&0xfb | byte(input)&0x4 },
	"setBit4":          func(input int, val byte) byte { return val&0xef | byte(input)&0x10 },
	"setBit2":          func(input int, val byte) byte { return val&0xbf | byte(input)&0x40 },
	"setBit1":          func(input int, val byte) byte { return val&0x7f | byte(input)&0x80 },
	"setOpMode":        setOperationMode,
	"temp2hex":         temp2hex,
	"demand2hex":       demand2hex,
}

func temp2hex(temp int, _ byte) byte {
	var hextemp byte = 0

	if temp >= 120 {
		hextemp = 0
	} else if temp <= -78 {
		hextemp = 255
	} else {
		const Uref float64 = 255
		const constant float64 = 3695
		const R25 float64 = 6340
		const T25 float64 = 25
		const Rf float64 = 6480
		const K float64 = 273.15
		var RT float64 = R25 * math.Exp(constant*(1/(float64(temp)+K)-1/(T25+K)))
		hextemp = byte(Uref * (RT / (Rf + RT)))
	}
	return hextemp
}

func demand2hex(demand int, _ byte) byte {
	var hexdemand byte = 0

	const min = 43 - 5 // 0% in hex
	const max = 234    // 100% in hex

	if demand >= 100 {
		hexdemand = max
	} else if demand <= 5 {
		hexdemand = min + 5
	} else {
		const a float64 = (max - min) / 100.
		hexdemand = byte(a*float64(demand) + min)
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
