package codec

import (
	"testing"

	"github.com/rondoval/GoHeishaMon/topics"
)

func TestEncode(t *testing.T) {
	topics := topics.LoadTopics("../../files/topics.yaml", "TestDevice", topics.Main)

	command := []byte{0xf1, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	holi, _ := topics.Lookup("Holiday_Mode_State")
	holi.UpdateValue("Off")
	encode(holi, command)
	if command[5] != 16 {
		t.Error("Holiday_Mode_State encode error")
	}

	holi.UpdateValue("Scheduled")
	encode(holi, command)
	if command[5] != 32 {
		t.Error("Holiday_Mode_State encode error")
	}

	holi.UpdateValue("Active")
	encode(holi, command)
	if command[5] != 48 {
		t.Error("Holiday_Mode_State encode error")
	}

	dhw, _ := topics.Lookup("DHW_Heat_Delta")
	dhw.UpdateValue("-5")
	encode(dhw, command)
	if command[99] != 123 {
		t.Error("DHW_Heat_Delta encode failed")
	}
}
