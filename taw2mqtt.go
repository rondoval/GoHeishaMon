package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/rs/xid"
	"go.bug.st/serial"
)

var panasonicQuery []byte = []byte{0x71, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

const panasonicQuerySize int = 110

//should be the same number
const numberOfTopics int = 95

var allTopics [95]topicData
var mqttKeepalive time.Duration
var commandsToSend map[xid.ID][]byte
var gpio map[string]string
var actData [92]string
var config configStruct
var sending bool
var serialPort serial.Port
var err error
var goodreads float64
var totalreads float64
var readpercentage float64
var switchTopics map[string]autoDiscoverStruct
var climateTopics map[string]autoDiscoverStruct

type commandStruct struct {
	value  [128]byte
	length int
}

type topicData struct {
	TopicNumber        int
	TopicName          string
	TopicBit           int
	TopicFunction      string
	TopicUnit          string
	TopicA2M           string
	TopicType          string
	TopicDisplayUnit   string
	TopicValueTemplate string
}

type configStruct struct {
	Readonly               bool
	Loghex                 bool
	Device                 string
	ReadInterval           int
	MqttServer             string
	MqttPort               string
	MqttLogin              string
	Aquarea2mqttCompatible bool
	MqttTopicBase          string
	MqttSetBase            string
	Aquarea2mqttPumpID     string
	MqttPass               string
	MqttClientID           string
	MqttKeepalive          int
	ForceRefreshTime       int
	EnableCommand          bool
	SleepAfterCommand      int
	HAAutoDiscover         bool
}

var configfile string

func updatePassword() bool {
	_, err = os.Stat("/mnt/usb/GoHeishaMonPassword.new")
	if err != nil {
		return true
	}
	_, _ = exec.Command("chmod", "+x", "/root/pass.sh").Output()
	dat, _ := ioutil.ReadFile("/mnt/usb/GoHeishaMonPassword.new")
	fmt.Printf("updejtuje haslo na: %s", string(dat))
	o, err := exec.Command("/root/pass.sh", string(dat)).Output()
	if err != nil {
		fmt.Println(err)
		fmt.Println(o)

		return false
	}
	fmt.Println(o)

	_, _ = exec.Command("/bin/rm", "/mnt/usb/GoHeishaMonPassword.new").Output()

	return true
}

func publishTopicsToAutoDiscover(mclient mqtt.Client, token mqtt.Token) {
	for k, v := range allTopics {
		var m autoDiscoverStruct
		m.UID = fmt.Sprintf("Aquarea-%s-%d", config.MqttLogin, k)
		if v.TopicType == "" {
			v.TopicType = "sensor"
		}
		m.ValueTemplate = v.TopicValueTemplate

		m.UnitOfM = v.TopicDisplayUnit

		if v.TopicType == "binary_sensor" {
			m.UnitOfM = ""
			m.PayloadOn = "1"
			m.PayloadOff = "0"
			m.ValueTemplate = `{{ value }}`
		}
		if v.TopicDisplayUnit == "°C" {
			m.DeviceClass = "temperature"
		}
		if v.TopicDisplayUnit == "W" {
			m.DeviceClass = "power"
		}
		m.StateTopic = fmt.Sprintf("%s/%s", config.MqttTopicBase, v.TopicName)
		m.Name = fmt.Sprintf("TEST-%s", v.TopicName)
		topicValue, err := json.Marshal(m)
		//v.TopicType = "sensor"

		fmt.Println(err)
		TOP := fmt.Sprintf("%s/%s/%s/config", config.MqttTopicBase, v.TopicType, strings.ReplaceAll(m.Name, " ", "_"))
		fmt.Println("Publikuje do ", TOP, "warosc", string(topicValue))
		token = mclient.Publish(TOP, byte(0), false, topicValue)
		if token.Wait() && token.Error() != nil {
			fmt.Printf("Fail to publish, %v", token.Error())
		}

	}

	for _, vs := range switchTopics {
		if vs.ValueTemplate == "" {
			vs.PayloadOff = "0"
			vs.PayloadOn = "1"
		}
		vs.Optimistic = "true"
		topicValue, err := json.Marshal(vs)

		fmt.Println(err)
		TOP := fmt.Sprintf("%s/%s/%s/config", config.MqttTopicBase, "switch", strings.ReplaceAll(vs.Name, " ", "_"))
		fmt.Println("Publikuje do ", TOP, "warosc", string(topicValue))
		token = mclient.Publish(TOP, byte(0), false, topicValue)
		if token.Wait() && token.Error() != nil {
			fmt.Printf("Fail to publish, %v", token.Error())
		}
	}

}

type autoDiscoverStruct struct {
	DeviceClass   string `json:"device_class,omitempty"`
	Name          string `json:"name,omitempty"`
	StateTopic    string `json:"state_topic,omitempty"`
	UnitOfM       string `json:"unit_of_measurement,omitempty"`
	ValueTemplate string `json:"value_template,omitempty"`
	CommandTopic  string `json:"command_topic,omitempty"`
	UID           string `json:"unique_id,omitempty"`
	PayloadOn     string `json:"payload_on,omitempty"`
	PayloadOff    string `json:"payload_off,omitempty"`
	Optimistic    string `json:"optimistic,omitempty"`
	StateON       string `json:"state_on,omitempty"`
	StateOff      string `json:"state_off,omitempty"`
}

func main() {
	switchTopics = make(map[string]autoDiscoverStruct)

	flag.Parse()
	if runtime.GOOS != "windows" {
		//	go UpdateGPIOStat()
		configfile = "/etc/gh/config"

	} else {
		configfile = "config"

	}
	_, err := os.Stat(configfile)
	if err != nil {
		fmt.Printf("Config file is missing: %s ", configfile)
		updateConfig(configfile)
	}
	go updateConfigLoop(configfile)
	c1 := make(chan bool, 1)
	go clearActData()
	commandsToSend = make(map[xid.ID][]byte)
	var in int
	config = readConfig()

	ports, err := serial.GetPortsList()
	if err != nil {
		log.Fatal(err)
	}
	if len(ports) == 0 {
		log.Fatal("No serial ports found!")
	}
	for _, port := range ports {
		fmt.Printf("Found port: %v\n", port)
	}
	mode := &serial.Mode{
		BaudRate: 9600,
		Parity:   serial.EvenParity,
		DataBits: 8,
		StopBits: serial.OneStopBit,
	}
	serialPort, err = serial.Open(config.Device, mode)
	if err != nil {
		fmt.Println(err)
	}
	PoolInterval := time.Second * time.Duration(config.ReadInterval)
	parseTopicList3()
	mqttKeepalive = time.Second * time.Duration(config.MqttKeepalive)
	MC, MT := makeMQTTConn()
	if config.HAAutoDiscover == true {
		publishTopicsToAutoDiscover(MC, MT)
	}
	for {
		if MC.IsConnected() != true {
			MC, MT = makeMQTTConn()
		}
		if len(commandsToSend) > 0 {
			fmt.Println("jest wiecej niz jedna komenda tj", len(commandsToSend))
			in = 1
			for key, value := range commandsToSend {
				if in == 1 {

					sendCommand(value, len(value))
					delete(commandsToSend, key)
					in++
					time.Sleep(time.Second * time.Duration(config.SleepAfterCommand))

				} else {
					fmt.Println("numer komenty  ", in, " jest za duzy zrobie to w nastepnym cyklu")
					break
				}
				fmt.Println("koncze range po tablicy z komendami ")

			}

		} else {
			sendCommand(panasonicQuery, panasonicQuerySize)
		}
		go func() {
			tbool := readSerial(MC, MT)
			c1 <- tbool
		}()

		select {
		case res := <-c1:
			fmt.Println("read ma status", res)
		case <-time.After(5 * time.Second):
			fmt.Println("out of time for read :(")
		}

		time.Sleep(PoolInterval)

	}

}

func clearActData() {
	for {
		time.Sleep(time.Second * time.Duration(config.ForceRefreshTime))
		for k := range actData {
			actData[k] = "nil" //funny i know ;)
		}

	}
}

func makeMQTTConn() (mqtt.Client, mqtt.Token) {
	opts := mqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("%s://%s:%s", "tcp", config.MqttServer, config.MqttPort))
	opts.SetPassword(config.MqttPass)
	opts.SetUsername(config.MqttLogin)
	opts.SetClientID(config.MqttClientID)
	opts.SetWill(config.MqttSetBase+"/LWT", "Offline", 1, true)
	opts.SetKeepAlive(mqttKeepalive)
	opts.SetOnConnectHandler(startsub)
	opts.SetConnectionLostHandler(connLostHandler)

	// connect to broker
	client := mqtt.NewClient(opts)
	//defer client.Disconnect(uint(2))

	token := client.Connect()
	if token.Wait() && token.Error() != nil {
		fmt.Printf("Fail to connect broker, %v", token.Error())
	}
	return client, token
}

func connLostHandler(c mqtt.Client, err error) {
	fmt.Printf("Connection lost, reason: %v\n", err)

	//Perform additional action...
}

func makeSwitchTopic(name string, state string) {
	var t autoDiscoverStruct
	t.Name = fmt.Sprintf("TEST-%s", name)
	t.StateTopic = config.MqttTopicBase + "/" + state
	t.CommandTopic = config.MqttSetBase + "/" + name
	t.UID = fmt.Sprintf("Aquarea-%s-%s", config.MqttLogin, t.Name)
	switchTopics[name] = t

}
func startsub(c mqtt.Client) {
	var t autoDiscoverStruct
	c.Subscribe("aquarea/+/+/set", 2, handleMSGfromMQTT)
	c.Subscribe(config.MqttSetBase+"/SetHeatpump", 2, handleSetHeatpump)
	makeSwitchTopic("SetHeatpump", "Heatpump_State")
	c.Subscribe(config.MqttSetBase+"/SetQuietMode", 2, handleSetQuietMode)
	c.Subscribe(config.MqttSetBase+"/SetZ1HeatRequestTemperature", 2, handleSetZ1HeatRequestTemperature)
	c.Subscribe(config.MqttSetBase+"/SetZ1CoolRequestTemperature", 2, handleSetZ1CoolRequestTemperature)
	c.Subscribe(config.MqttSetBase+"/SetZ2HeatRequestTemperature", 2, handleSetZ2HeatRequestTemperature)
	c.Subscribe(config.MqttSetBase+"/SetZ2CoolRequestTemperature", 2, handleSetZ2CoolRequestTemperature)
	c.Subscribe(config.MqttSetBase+"/SetOperationMode", 2, handleSetOperationMode)
	c.Subscribe(config.MqttSetBase+"/SetForceDHW", 2, handleSetForceDHW)
	makeSwitchTopic("SetForceDHW", "Force_DHW_State")
	c.Subscribe(config.MqttSetBase+"/SetForceDefrost", 2, handleSetForceDefrost)
	makeSwitchTopic("SetForceDefrost", "Defrosting_State")
	c.Subscribe(config.MqttSetBase+"/SetForceSterilization", 2, handleSetForceSterilization)
	makeSwitchTopic("SetForceSterilization", "Sterilization_State")
	c.Subscribe(config.MqttSetBase+"/SetHolidayMode", 2, handleSetHolidayMode)
	makeSwitchTopic("SetHolidayMode", "Holiday_Mode_State")
	c.Subscribe(config.MqttSetBase+"/SetPowerfulMode", 2, handleSetPowerfulMode)

	t.Name = "TEST-SetPowerfulMode-30min"
	t.CommandTopic = config.MqttSetBase + "/SetPowerfulMode"
	t.StateTopic = config.MqttTopicBase + "/Powerful_Mode_Time"
	t.UID = fmt.Sprintf("Aquarea-%s-%s", config.MqttLogin, t.Name)
	t.PayloadOn = "1"
	t.StateON = "on"
	t.StateOff = "off"
	t.ValueTemplate = `{%- if value == "1" -%} on {%- else -%} off {%- endif -%}`
	switchTopics["SetPowerfulMode1"] = t
	t = autoDiscoverStruct{}
	t.Name = "TEST-SetPowerfulMode-60min"
	t.CommandTopic = config.MqttSetBase + "/SetPowerfulMode"
	t.StateTopic = config.MqttTopicBase + "/Powerful_Mode_Time"
	t.UID = fmt.Sprintf("Aquarea-%s-%s", config.MqttLogin, t.Name)
	t.PayloadOn = "2"
	t.StateON = "on"
	t.StateOff = "off"
	t.ValueTemplate = `{%- if value == "2" -%} on {%- else -%} off {%- endif -%}`
	switchTopics["SetPowerfulMode2"] = t
	t = autoDiscoverStruct{}
	t.Name = "TEST-SetPowerfulMode-90min"
	t.CommandTopic = config.MqttSetBase + "/SetPowerfulMode"
	t.StateTopic = config.MqttTopicBase + "/Powerful_Mode_Time"
	t.UID = fmt.Sprintf("Aquarea-%s-%s", config.MqttLogin, t.Name)
	t.PayloadOn = "3"
	t.StateON = "on"
	t.StateOff = "off"
	t.ValueTemplate = `{%- if value == "3" -%} on {%- else -%} off {%- endif -%}`
	switchTopics["SetPowerfulMode3"] = t
	t = autoDiscoverStruct{}

	c.Subscribe(config.MqttSetBase+"/SetDHWTemp", 2, handleSetDHWTemp)
	c.Subscribe(config.MqttSetBase+"/SendRawValue", 2, handleSendRawValue)
	if config.EnableCommand == true {
		c.Subscribe(config.MqttSetBase+"/OSCommand", 2, handleOSCommand)
	}

	//Perform additional action...
}
