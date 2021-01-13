package main

import (
	"encoding/json"
	"log"
	"math"
	"strconv"
)

const setCmdLen = 110

var panasonicSetCommand = [setCmdLen]byte{0xf1, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

var mainCommandMap = map[string]func(byte) (byte, int){
	// set heatpump state to on by sending 1
	"SetHeatpump": func(val byte) (byte, int) { return val + 1, 4 },
	// set pump state to on by sending 1
	"SetPump": func(val byte) (byte, int) { return (val + 1) << 4, 4 },
	// set pump speed
	"SetPumpSpeed": func(val byte) (byte, int) { return val + 1, 45 },
	// set 0 for Off mode, set 1 for Quiet mode 1, set 2 for Quiet mode 2, set 3 for Quiet mode 3
	"SetQuietMode": func(val byte) (byte, int) { return (val + 1) * 8, 7 },
	// z1 heat request temp -  set from -5 to 5 to get same temperature shift point or set direct temp
	"SetZ1HeatRequestTemperature": func(val byte) (byte, int) { return val + 128, 38 },
	// z1 cool request temp -  set from -5 to 5 to get same temperature shift point or set direct temp
	"SetZ1CoolRequestTemperature": func(val byte) (byte, int) { return val + 128, 39 },
	// z2 heat request temp -  set from -5 to 5 to get same temperature shift point or set direct temp
	"SetZ2HeatRequestTemperature": func(val byte) (byte, int) { return val + 128, 40 },
	// z2 cool request temp -  set from -5 to 5 to get same temperature shift point or set direct temp
	"SetZ2CoolRequestTemperature": func(val byte) (byte, int) { return val + 128, 41 },
	// set mode to force DHW by sending 1
	"SetForceDHW": func(val byte) (byte, int) { return (val + 1) << 6, 4 },
	// set mode to force defrost  by sending 1
	"SetForceDefrost": func(val byte) (byte, int) { return (val & 1) << 1, 8 },
	// set mode to force sterilization by sending 1
	"SetForceSterilization": func(val byte) (byte, int) { return (val & 1) << 2, 8 },
	// set Holiday mode by sending 1, off will be 0
	"SetHolidayMode": func(val byte) (byte, int) { return (val + 1) << 4, 5 },
	// set Powerful mode by sending 0 = off, 1 for 30min, 2 for 60min, 3 for 90 min
	"SetPowerfulMode": func(val byte) (byte, int) { return val + 73, 7 },
	// set Heat pump operation mode  0 = heat only, 1 = cool only, 2 = Auto, 3 = DHW only, 4 = Heat+DHW, 5 = Cool+DHW, 6 = Auto + DHW
	"SetOperationMode": setOperationMode,
	// set DHW temperature by sending desired temperature between 40C-75C
	"SetDHWTemp": func(val byte) (byte, int) { return val + 128, 42 },
	// set zones to active
	"SetZones":          func(val byte) (byte, int) { return (val + 1) << 6, 6 },
	"SetFloorHeatDelta": func(val byte) (byte, int) { return val + 128, 84 },
	"SetFloorCoolDelta": func(val byte) (byte, int) { return val + 128, 94 },
	"SetDHWHeatDelta":   func(val byte) (byte, int) { return val + 128, 99 },
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

func setOperationMode(val byte) (data byte, index int) {
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
	return data, 6
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
