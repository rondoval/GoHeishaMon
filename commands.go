package main

import (
	"math"
)

const setCmdLen = 110

var panasonicSetCommand = [setCmdLen]byte{0xf1, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

var encodeInt = map[string]func(int) byte{
	"setIntDiv50Plus1": func(input int) byte { return byte(input/50) + 1 },
	"setIntDiv30Plus1": func(input int) byte { return byte(input/30) + 1 },
	"setIntDiv10Plus1": func(input int) byte { return byte(input/10) + 1 },
	"setIntPlus128":    func(input int) byte { return byte(input) + 128 },
	"setIntPlus1":      func(input int) byte { return byte(input) + 1 },
	"setRight3bits":    func(input int) byte { return byte(input+1) & 7 },
	"setLeft5bits":     func(input int) byte { return byte(input+1) & 31 << 3 },
	"setBit3and4and5":  func(input int) byte { return byte(input+1) & 7 << 3 },
	"setBit7and8":      func(input int) byte { return byte(input+1) & 3 },
	"setBit5and6":      func(input int) byte { return byte(input+1) & 3 << 2 },
	"setBit3and4":      func(input int) byte { return byte(input+1) & 3 << 4 },
	"setBit1and2":      func(input int) byte { return byte(input+1) & 3 << 6 },
	"setBit7":          func(input int) byte { return byte(input) & 1 << 1 },
	"setBit6":          func(input int) byte { return byte(input) & 1 << 2 },
	"setOpMode":        setOperationMode,
}

var optionCommandMapByte = map[string]func(byte){
	"SetHeatCoolMode":             func(val byte) { optionalPCBQuery[6] = (optionalPCBQuery[6] & ^(byte(0b1) << 7)) | (val & 1 << 7) },
	"SetCompressorState":          func(val byte) { optionalPCBQuery[6] = (optionalPCBQuery[6] & ^(byte(0b1) << 6)) | (val & 1 << 6) },
	"SetSmartGridMode":            func(val byte) { optionalPCBQuery[6] = (optionalPCBQuery[6] & ^(byte(0b11) << 4)) | (val & 3 << 4) },
	"SetExternalThermostat1State": func(val byte) { optionalPCBQuery[6] = (optionalPCBQuery[6] & ^(byte(0b11) << 2)) | (val & 3 << 2) },
	"SetExternalThermostat2State": func(val byte) { optionalPCBQuery[6] = (optionalPCBQuery[6] & ^byte(0b11)) | (val & 3) },
	"SetDemandControl":            func(val byte) { optionalPCBQuery[14] = val },
}

var optionCommandMapFloat = map[string]func(float64){
	"SetPoolTemp":    func(temp float64) { optionalPCBQuery[7] = temp2hex(temp) },
	"SetBufferTemp":  func(temp float64) { optionalPCBQuery[8] = temp2hex(temp) },
	"SetZ1RoomTemp":  func(temp float64) { optionalPCBQuery[10] = temp2hex(temp) },
	"SetZ1WaterTemp": func(temp float64) { optionalPCBQuery[16] = temp2hex(temp) },
	"SetZ2RoomTemp":  func(temp float64) { optionalPCBQuery[11] = temp2hex(temp) },
	"SetZ2WaterTemp": func(temp float64) { optionalPCBQuery[15] = temp2hex(temp) },
	"SetSolarTemp":   func(temp float64) { optionalPCBQuery[13] = temp2hex(temp) },
}

func temp2hex(temp float64) byte {
	var hextemp byte = 0

	if temp > 120 {
		hextemp = 0
	} else if temp < -78 {
		hextemp = 255
	} else {
		const Uref float64 = 255
		const constant float64 = 3695
		const R25 float64 = 6340
		const T25 float64 = 25
		const Rf float64 = 6480
		const K float64 = 273.15
		var RT float64 = R25 * math.Exp(constant*(1/(temp+K)-1/(T25+K)))
		hextemp = byte(Uref * (RT / (Rf + RT)))
	}
	return hextemp
}

func setOperationMode(val int) (data byte) {
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
