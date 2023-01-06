package topics

import (
	"testing"
)

func TestLoadMain(t *testing.T) {
	topics := LoadTopics("../../files/topics.yaml", "TestDevice", Main)
	if topics.Kind() != Main {
		t.Error("Wrong kind")
	}
	if topics.DeviceName() != "TestDevice" {
		t.Error("Name not set")
	}

	if val, _ := topics.Lookup("Force_DHW_State"); !val.Writable() {
		t.Error("Writable attribute wrong")
	}

	inlet, _ := topics.Lookup("Main_Inlet_Temp")

	if inlet.Writable() {
		t.Error("Writable attribute wrong")
	}

	if len(inlet.Codec) != 2 {
		t.Error("Codec entries missing")
	}
}
