package codec

import (
	"testing"

	"github.com/rondoval/GoHeishaMon/topics"
)

func TestDecode(t *testing.T) {
	topicData := topics.LoadTopics("../../files/topics.yaml", "TestDevice", topics.Main)

	command := []byte{
		0x71, 0xC8, 0x01, 0x10, 0x56, 0x55, 0x61, 0x49, 0x00, 0x55,
		0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x59, 0x15, 0x11, 0x55, 0x56, 0x16, 0x55, 0x55, 0x55, 0x29,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x7B, 0x85,
		0x80, 0x80, 0xB4, 0x71, 0x71, 0x83, 0x99, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80, 0x85,
		0x1F, 0x8A, 0x85, 0x85, 0xD0, 0x7B, 0x78, 0x1F, 0x7E, 0x1F,
		0x1F, 0x79, 0x79, 0x8D, 0x8D, 0xB7, 0xA3, 0x80, 0x8F, 0xB7,
		0xA3, 0x7B, 0x8F, 0x96, 0x8A, 0x78, 0x8F, 0x8A, 0x94, 0x9E,
		0x8F, 0x8A, 0x94, 0x9E, 0x85, 0x8F, 0x8A, 0x0B, 0x5B, 0x78,
		0xC1, 0x1F, 0x7E, 0x7C, 0x1F, 0x7C, 0x7E, 0x00, 0x00, 0x00,
		0x55, 0x65, 0x55, 0x21, 0x87, 0x15, 0x55, 0x05, 0x0C, 0x12,
		0x65, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x32,
		0xD4, 0x0B, 0x87, 0x84, 0x73, 0x90, 0x0C, 0x84, 0x84, 0x94,
		0x00, 0xAF, 0x98, 0x94, 0x94, 0x32, 0x32, 0x9E, 0xA3, 0x32,
		0x32, 0x32, 0x7B, 0x9E, 0x95, 0x95, 0x97, 0x95, 0x97, 0x96,
		0x95, 0x96, 0x9C, 0x48, 0x01, 0x01, 0x01, 0x00, 0x00, 0x22,
		0x00, 0x01, 0x01, 0x01, 0x01, 0x79, 0x79, 0x01, 0x01, 0x8C,
		0x03, 0x00, 0xD4, 0x01, 0x00, 0x01, 0x00, 0x00, 0x01, 0x00,
		0x00, 0x0A, 0x02, 0x01, 0x01, 0x01, 0x01, 0x01, 0x01, 0x02,
		0x00, 0x00,
	}
	command[5] = 32    // Scheduled
	command[99] = 123  // -5
	command[143] = 150 // inlet 22
	command[118] = 3   // inlet 0.5
	command[169] = 57  // pump flow 0.22
	command[170] = 25  // pump flow 25
	command[22] = 0x14 // Z1 - thermistor, Z2 - water temp

	Decode(topicData, command, true)
	{
		holi, _ := topicData.Lookup("Holiday_Mode_State")
		t.Logf("Holiday_Mode_State %s", holi.CurrentValue())
		if holi.CurrentValue() != "Scheduled" {
			t.Error("Holiday_Mode_State decode error")
		}

		dhw, _ := topicData.Lookup("DHW_Heat_Delta")
		t.Logf("DHW_Heat_Delta %s", dhw.CurrentValue())
		if dhw.CurrentValue() != "-5" {
			t.Errorf("DHW_Heat_Delta decode error: %s", dhw.CurrentValue())
		}

		inlet, _ := topicData.Lookup("Main_Inlet_Temp")
		t.Logf("Main_Inlet_Temp %s", inlet.CurrentValue())
		if inlet.CurrentValue() != "22.50" {
			t.Error("Wrong inlet temp")
		}

		pumpFlow, _ := topicData.Lookup("Pump_Flow")
		t.Logf("Pump_Flow %s", pumpFlow.CurrentValue())
		if pumpFlow.CurrentValue() != "25.22" {
			t.Error("Pump_Flow decode error")
		}

		z1set, _ := topicData.Lookup("Z1_Sensor_Settings")
		t.Logf("Z1_Sensor_Settings %s", z1set.CurrentValue())
		if z1set.CurrentValue() != "Thermistor" {
			t.Error("Z1_Sensor_Settings decode error")
		}

		z2set, _ := topicData.Lookup("Z2_Sensor_Settings")
		t.Logf("Z2_Sensor_Settings %s", z2set.CurrentValue())
		if z2set.CurrentValue() != "Water temperature" {
			t.Error("Z2_Sensor_Settings decode error")
		}

		model, _ := topicData.Lookup("Heat_Pump_Model")
		t.Logf("Heat_Pump_Model %s", model.CurrentValue())
		if model.CurrentValue() != "IDU: Monoblock ODU: WH-MXC09J3E8" {
			t.Error("Unexpected heat pump model")
		}
	}

	// test raw values
	Decode(topicData, command, false)
	{
		holi, _ := topicData.Lookup("Holiday_Mode_State")
		t.Logf("Holiday_Mode_State %s", holi.CurrentValue())
		if holi.CurrentValue() != "1" {
			t.Error("Holiday_Mode_State decode error")
		}

		dhw, _ := topicData.Lookup("DHW_Heat_Delta")
		t.Logf("DHW_Heat_Delta %s", dhw.CurrentValue())
		if dhw.CurrentValue() != "-5" {
			t.Errorf("DHW_Heat_Delta decode error: %s", dhw.CurrentValue())
		}

		inlet, _ := topicData.Lookup("Main_Inlet_Temp")
		t.Logf("Main_Inlet_Temp %s", inlet.CurrentValue())
		if inlet.CurrentValue() != "22.50" {
			t.Error("Wrong inlet temp")
		}

		pumpFlow, _ := topicData.Lookup("Pump_Flow")
		t.Logf("Pump_Flow %s", pumpFlow.CurrentValue())
		if pumpFlow.CurrentValue() != "25.22" {
			t.Error("Pump_Flow decode error")
		}

		z1set, _ := topicData.Lookup("Z1_Sensor_Settings")
		t.Logf("Z1_Sensor_Settings %s", z1set.CurrentValue())
		if z1set.CurrentValue() != "3" {
			t.Error("Z1_Sensor_Settings decode error")
		}

		z2set, _ := topicData.Lookup("Z2_Sensor_Settings")
		t.Logf("Z2_Sensor_Settings %s", z2set.CurrentValue())
		if z2set.CurrentValue() != "0" {
			t.Error("Z2_Sensor_Settings decode error")
		}

		model, _ := topicData.Lookup("Heat_Pump_Model")
		t.Logf("Heat_Pump_Model %s", model.CurrentValue())
		if model.CurrentValue() != "IDU: Monoblock ODU: WH-MXC09J3E8" {
			t.Error("Unexpected heat pump model")
		}
	}
}
