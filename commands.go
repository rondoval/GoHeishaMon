package main

import (
	"encoding/json"
	"log"
	"math"
	"strconv"
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
		data = 18
	case 1:
		data = 19
	case 2:
		data = 24
	case 3:
		data = 33
	case 4:
		data = 34
	case 5:
		data = 35
	case 6:
		data = 40
	default:
		data = 0
	}
	return data
}

func setCurves(msg string) ([110]byte, error) {
	type tempRange struct {
		high string
		low  string
	}
	type tempCurve struct {
		target  tempRange
		outside tempRange
	}
	type curves struct {
		zone1 struct {
			heat tempCurve
			cool tempCurve
		}
		zone2 struct {
			heat tempCurve
			cool tempCurve
		}
	}

	command := panasonicSetCommand
	var n curves

	err := json.Unmarshal([]byte(msg), &n)
	if err != nil {
		log.Println("SetCurves JSON decode failed!")
		return command, err
	}

	log.Println("SetCurves JSON received ok")

	if n.zone1.heat.target.high != "" {
		v, _ := strconv.Atoi(n.zone1.heat.target.high)
		command[75] = byte(v + 128)
	}
	if n.zone1.heat.target.low != "" {
		v, _ := strconv.Atoi(n.zone1.heat.target.low)
		command[76] = byte(v + 128)
	}
	if n.zone1.heat.outside.low != "" {
		v, _ := strconv.Atoi(n.zone1.heat.outside.low)
		command[77] = byte(v + 128)
	}
	if n.zone1.heat.outside.high != "" {
		v, _ := strconv.Atoi(n.zone1.heat.outside.high)
		command[78] = byte(v + 128)
	}
	if n.zone2.heat.target.high != "" {
		v, _ := strconv.Atoi(n.zone2.heat.target.high)
		command[79] = byte(v + 128)
	}
	if n.zone2.heat.target.low != "" {
		v, _ := strconv.Atoi(n.zone2.heat.target.low)
		command[80] = byte(v + 128)
	}
	if n.zone2.heat.outside.low != "" {
		v, _ := strconv.Atoi(n.zone2.heat.outside.low)
		command[81] = byte(v + 128)
	}
	if n.zone2.heat.outside.high != "" {
		v, _ := strconv.Atoi(n.zone2.heat.target.high)
		command[82] = byte(v + 128)
	}
	if n.zone1.cool.target.high != "" {
		v, _ := strconv.Atoi(n.zone1.cool.target.high)
		command[86] = byte(v + 128)
	}
	if n.zone1.cool.target.low != "" {
		v, _ := strconv.Atoi(n.zone1.cool.target.low)
		command[87] = byte(v + 128)
	}
	if n.zone1.cool.outside.low != "" {
		v, _ := strconv.Atoi(n.zone1.cool.outside.low)
		command[88] = byte(v + 128)
	}
	if n.zone1.cool.outside.high != "" {
		v, _ := strconv.Atoi(n.zone1.cool.outside.high)
		command[89] = byte(v + 128)
	}
	if n.zone2.cool.target.high != "" {
		v, _ := strconv.Atoi(n.zone2.cool.target.high)
		command[90] = byte(v + 128)
	}
	if n.zone2.cool.target.low != "" {
		v, _ := strconv.Atoi(n.zone2.cool.target.low)
		command[91] = byte(v + 128)
	}
	if n.zone2.cool.outside.low != "" {
		v, _ := strconv.Atoi(n.zone2.cool.outside.low)
		command[92] = byte(v + 128)
	}
	if n.zone2.cool.outside.high != "" {
		v, _ := strconv.Atoi(n.zone2.cool.outside.high)
		command[93] = byte(v + 128)
	}
	return command, nil
}
