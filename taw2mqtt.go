package main

import (
	"bufio"
	"crypto/md5"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
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

var cfgfile *string
var topicfile *string
var configfile string

func fileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func setGPIODebug() {
	err := ioutil.WriteFile("/sys/class/gpio/export", []byte("2"), 0200)
	err = ioutil.WriteFile("/sys/class/gpio/export", []byte("3"), 0200)
	err = ioutil.WriteFile("/sys/class/gpio/export", []byte("13"), 0200)
	err = ioutil.WriteFile("/sys/class/gpio/export", []byte("15"), 0200)
	err = ioutil.WriteFile("/sys/class/gpio/export", []byte("10"), 0200)
	err = ioutil.WriteFile("/sys/class/gpio/export", []byte("0"), 0200)
	err = ioutil.WriteFile("/sys/class/gpio/export", []byte("1"), 0200)
	err = ioutil.WriteFile("/sys/class/gpio/export", []byte("16"), 0200)

	if err != nil {
		fmt.Println(err.Error())
	}
}

func getGPIOStatus() {
	readFile, err := os.Open("/sys/kernel/debug/gpio")
	//readFile, err := os.Open("FakeKernel.txt")
	if err != nil {
		log.Fatalf("failed to open file: %s", err)
	}

	fileScanner := bufio.NewScanner(readFile)
	fileScanner.Split(bufio.ScanLines)
	var fileTextLines []string

	for fileScanner.Scan() {
		fileTextLines = append(fileTextLines, fileScanner.Text())
	}

	readFile.Close()

	for _, eachline := range fileTextLines {
		s := strings.Fields(eachline)
		if len(s) > 3 {
			gpio[s[0]] = s[4]
		}

	}
	if len(gpio) > 1 {
		fmt.Println(gpio)
		if gpio["gpio-0"] == "lo" && gpio["gpio-1"] == "lo" && gpio["gpio-16"] == "hi" {
			err = ioutil.WriteFile("/sys/class/gpio/gpio2/direction", []byte("high"), 644)
			err = ioutil.WriteFile("/sys/class/gpio/gpio13/direction", []byte("high"), 644)
			err = ioutil.WriteFile("/sys/class/gpio/gpio15/direction", []byte("high"), 644)
		}
		if gpio["gpio-0"] == "hi" || gpio["gpio-1"] == "hi" || gpio["gpio-16"] == "lo" {
			err = ioutil.WriteFile("/sys/class/gpio/gpio2/direction", []byte("high"), 644)
			err = ioutil.WriteFile("/sys/class/gpio/gpio13/direction", []byte("low"), 644)
			err = ioutil.WriteFile("/sys/class/gpio/gpio15/direction", []byte("low"), 644)
		}
		if gpio["gpio-0"] == "hi" && gpio["gpio-1"] == "hi" {
			err = ioutil.WriteFile("/sys/class/gpio/gpio2/direction", []byte("low"), 644)
			err = ioutil.WriteFile("/sys/class/gpio/gpio13/direction", []byte("high"), 644)
			err = ioutil.WriteFile("/sys/class/gpio/gpio15/direction", []byte("high"), 644)
		}
		if gpio["gpio-0"] == "hi" && gpio["gpio-16"] == "lo" {
			err = ioutil.WriteFile("/sys/class/gpio/gpio2/direction", []byte("low"), 644)
			err = ioutil.WriteFile("/sys/class/gpio/gpio13/direction", []byte("high"), 644)
			err = ioutil.WriteFile("/sys/class/gpio/gpio15/direction", []byte("high"), 644)
		}
		if gpio["gpio-1"] == "hi" && gpio["gpio-16"] == "lo" {
			err = ioutil.WriteFile("/sys/class/gpio/gpio2/direction", []byte("low"), 644)
			err = ioutil.WriteFile("/sys/class/gpio/gpio13/direction", []byte("high"), 644)
			err = ioutil.WriteFile("/sys/class/gpio/gpio15/direction", []byte("high"), 644)
		}
		if gpio["gpio-0"] == "hi" && gpio["gpio-1"] == "hi" && gpio["gpio-16"] == "lo" {
			err = ioutil.WriteFile("/sys/class/gpio/gpio2/direction", []byte("low"), 644)
			err = ioutil.WriteFile("/sys/class/gpio/gpio13/direction", []byte("low"), 644)
			err = ioutil.WriteFile("/sys/class/gpio/gpio15/direction", []byte("high"), 644)
			cmd := exec.Command("fwupdate", "sw")
			out, err := cmd.CombinedOutput()
			fmt.Println(out)
			cmd = exec.Command("sync")
			out, err = cmd.CombinedOutput()
			fmt.Println(out)
			cmd = exec.Command("reboot")
			out, err = cmd.CombinedOutput()
			fmt.Println(out)
			if err != nil {
				fmt.Println(err)
			}

		}
		if gpio["gpio-10"] == "hi" {
			err := ioutil.WriteFile("/sys/class/gpio/gpio3/direction", []byte("low"), 644)
			if err != nil {
				fmt.Println(err)
			}

		}
		if gpio["gpio-10"] == "lo" {
			err := ioutil.WriteFile("/sys/class/gpio/gpio3/direction", []byte("high"), 644)
			if err != nil {
				fmt.Println(err)
			}

		}

	}
	if err != nil {
		fmt.Println(err)
	}
	time.Sleep(time.Nanosecond * 500000000)

}

func readConfig() configStruct {

	_, err := os.Stat(configfile)
	if err != nil {
		log.Fatal("Config file is missing: ", configfile)
	}

	var config configStruct
	if _, err := toml.DecodeFile(configfile, &config); err != nil {
		log.Fatal(err)
	}
	return config
}

func updateConfig(configfile string) bool {
	fmt.Printf("try to update configfile: %s", configfile)
	out, err := exec.Command("/usr/bin/usb_mount.sh").Output()
	if err != nil {
		fmt.Println(err.Error())
	}
	fmt.Println(out)
	_, err = os.Stat("/mnt/usb/GoHeishaMonConfig.new")
	if err != nil {
		_, _ = exec.Command("/usr/bin/usb_umount.sh").Output()
		return false
	}
	if getFileChecksum(configfile) != getFileChecksum("/mnt/usb/GoHeishaMonConfig.new") {
		fmt.Printf("checksum of configfile and new configfile diffrent: %s ", configfile)

		_, _ = exec.Command("/bin/cp", "/mnt/usb/GoHeishaMonConfig.new", configfile).Output()
		if err != nil {
			fmt.Printf("can't update configfile %s", configfile)
			return false
		}
		_, _ = exec.Command("sync").Output()

		_, _ = exec.Command("/usr/bin/usb_umount.sh").Output()
		_, _ = exec.Command("reboot").Output()
		return true
	}
	_, _ = exec.Command("/usr/bin/usb_umount.sh").Output()

	return true
}

func encodeTopicsToTOML(topnr int, data topicData) {
	f, err := os.Create(fmt.Sprintf("data/%d", topnr))
	if err != nil {
		// failed to create/open the file
		log.Fatal(err)
	}
	if err := toml.NewEncoder(f).Encode(data); err != nil {
		// failed to encode
		log.Fatal(err)
	}
	if err := f.Close(); err != nil {
		// failed to close the file
		log.Fatal(err)

	}

}
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

func getFileChecksum(f string) string {
	input := strings.NewReader(f)

	hash := md5.New()
	if _, err := io.Copy(hash, input); err != nil {
		log.Fatal(err)
	}
	sum := hash.Sum(nil)

	return fmt.Sprintf("%x\n", sum)

}

func updateConfigLoop(configfile string) {
	for {
		updateConfig(configfile)
		time.Sleep(time.Minute * 5)

	}
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
		//Topic_Value = []byte("")
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
		//Topic_Value = []byte("")

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

func updateGPIOStat() {

	// watcher := gpio.NewWatcher()
	// //watcher.AddPin(0)
	// watcher.AddPin(1)
	// watcher.AddPin(2)
	// watcher.AddPin(3)
	// watcher.AddPin(4)
	// watcher.AddPin(5)
	// watcher.AddPin(6)
	// watcher.AddPin(7)
	// watcher.AddPin(8)
	// watcher.AddPin(9)
	// watcher.AddPin(10)
	// watcher.AddPin(11)
	// watcher.AddPin(12)
	// watcher.AddPin(13)
	// watcher.AddPin(14)
	// watcher.AddPin(15)
	// watcher.AddPin(16)

	// defer watcher.Close()

	// go func() {
	// 	var v string
	// 	for {
	// 		pin, value := watcher.Watch()
	// 		if value == 1 {
	// 			v = "hi"
	// 		} else {
	// 			v = "lo"
	// 		}
	// 		GPIO[fmt.Sprintf("gpio-%d", pin)] = v
	// 		fmt.Printf("read %d from gpio %d\n", value, pin)
	// 	}
	// }()

	gpio = make(map[string]string)
	setGPIODebug()
	for {
		getGPIOStatus()
		//time.Sleep(time.Nanosecond * 500000000)
	}
}

func executeGPIOCommand() {
	for {
		var err error
		if len(gpio) > 1 {
			fmt.Println(gpio)
			if gpio["gpio-0"] == "lo" && gpio["gpio-1"] == "lo" && gpio["gpio-16"] == "hi" {
				err = ioutil.WriteFile("/sys/class/gpio/gpio2/direction", []byte("high"), 644)
				err = ioutil.WriteFile("/sys/class/gpio/gpio13/direction", []byte("high"), 644)
				err = ioutil.WriteFile("/sys/class/gpio/gpio15/direction", []byte("high"), 644)
			}
			if gpio["gpio-0"] == "hi" || gpio["gpio-1"] == "hi" || gpio["gpio-16"] == "lo" {
				err = ioutil.WriteFile("/sys/class/gpio/gpio2/direction", []byte("high"), 644)
				err = ioutil.WriteFile("/sys/class/gpio/gpio13/direction", []byte("low"), 644)
				err = ioutil.WriteFile("/sys/class/gpio/gpio15/direction", []byte("low"), 644)
			}
			if gpio["gpio-0"] == "hi" && gpio["gpio-1"] == "hi" {
				err = ioutil.WriteFile("/sys/class/gpio/gpio2/direction", []byte("low"), 644)
				err = ioutil.WriteFile("/sys/class/gpio/gpio13/direction", []byte("high"), 644)
				err = ioutil.WriteFile("/sys/class/gpio/gpio15/direction", []byte("high"), 644)
			}
			if gpio["gpio-0"] == "hi" && gpio["gpio-16"] == "lo" {
				err = ioutil.WriteFile("/sys/class/gpio/gpio2/direction", []byte("low"), 644)
				err = ioutil.WriteFile("/sys/class/gpio/gpio13/direction", []byte("high"), 644)
				err = ioutil.WriteFile("/sys/class/gpio/gpio15/direction", []byte("high"), 644)
			}
			if gpio["gpio-1"] == "hi" && gpio["gpio-16"] == "lo" {
				err = ioutil.WriteFile("/sys/class/gpio/gpio2/direction", []byte("low"), 644)
				err = ioutil.WriteFile("/sys/class/gpio/gpio13/direction", []byte("high"), 644)
				err = ioutil.WriteFile("/sys/class/gpio/gpio15/direction", []byte("high"), 644)
			}
			if gpio["gpio-0"] == "hi" && gpio["gpio-1"] == "hi" && gpio["gpio-16"] == "lo" {
				err = ioutil.WriteFile("/sys/class/gpio/gpio2/direction", []byte("low"), 644)
				err = ioutil.WriteFile("/sys/class/gpio/gpio13/direction", []byte("low"), 644)
				err = ioutil.WriteFile("/sys/class/gpio/gpio15/direction", []byte("high"), 644)
				cmd := exec.Command("fwupdate", "sw")
				out, err := cmd.CombinedOutput()
				fmt.Println(out)
				cmd = exec.Command("sync")
				out, err = cmd.CombinedOutput()
				fmt.Println(out)
				cmd = exec.Command("reboot")
				out, err = cmd.CombinedOutput()
				fmt.Println(out)
				fmt.Println(err)

			}
			if gpio["gpio-10"] == "hi" {
				err := ioutil.WriteFile("/sys/class/gpio/gpio3/direction", []byte("low"), 644)
				fmt.Println(err)

			}
			if gpio["gpio-10"] == "lo" {
				err := ioutil.WriteFile("/sys/class/gpio/gpio3/direction", []byte("high"), 644)
				fmt.Println(err)

			}

		}
		//time.Sleep(time.Nanosecond * 500000000)
		fmt.Println(err)

	}
}

func main() {
	switchTopics = make(map[string]autoDiscoverStruct)

	//	cfgfile = flag.String("c", "config", "a config file patch")
	//	topicfile = flag.String("t", "Topics.csv", "a topic file patch")
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
	// if config.Readonly != true {
	// 	log_message("Not sending this command. Heishamon in listen only mode! - this POC version don't support writing yet....")
	// 	os.Exit(0)
	// }
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

func handleMSGfromMQTT(mclient mqtt.Client, msg mqtt.Message) {

}

func remove(slice []string, s int) []string {
	return append(slice[:s], slice[s+1:]...)
}

func handleOSCommand(mclient mqtt.Client, msg mqtt.Message) {
	var cmd *exec.Cmd
	var out2 string
	s := strings.Split(string(msg.Payload()), " ")
	if len(s) < 2 {
		cmd = exec.Command(s[0])
	} else {
		cmd = exec.Command(s[0], s[1:]...)
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		// TODO: handle error more gracefully
		out2 = fmt.Sprintf("%s", err)
	}
	comout := fmt.Sprintf("%s - %s", out, out2)
	TOP := fmt.Sprintf("%s/OSCommand/out", config.MqttSetBase)
	fmt.Println("Publikuje do ", TOP, "warosc", string(comout))
	token := mclient.Publish(TOP, byte(0), false, comout)
	if token.Wait() && token.Error() != nil {
		fmt.Printf("Fail to publish, %v", token.Error())
	}

}

func handleSendRawValue(mclient mqtt.Client, msg mqtt.Message) {
	var command []byte
	cts := strings.TrimSpace(string(msg.Payload()))
	command, err = hex.DecodeString(cts)
	if err != nil {
		fmt.Println(err)
	}

	commandsToSend[xid.New()] = command
}

func handleSetOperationMode(mclient mqtt.Client, msg mqtt.Message) {
	var command []byte
	var setMode byte
	a, _ := strconv.Atoi(string(msg.Payload()))

	switch a {
	case 0:
		setMode = 82
	case 1:
		setMode = 83
	case 2:
		setMode = 89
	case 3:
		setMode = 33
	case 4:
		setMode = 98
	case 5:
		setMode = 99
	case 6:
		setMode = 104
	default:
		setMode = 0
	}

	fmt.Printf("set heat pump mode to  %d", setMode)
	command = []byte{0xf1, 0x6c, 0x01, 0x10, 0x00, 0x00, setMode, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if config.Loghex == true {
		logHex(command, len(command))
	}
	commandsToSend[xid.New()] = command
}

func handleSetDHWTemp(mclient mqtt.Client, msg mqtt.Message) {
	var command []byte
	var heatpumpState byte

	a, er := strconv.Atoi(string(msg.Payload()))
	if er != nil {
		f, _ := strconv.ParseFloat(string(msg.Payload()), 64)
		a = int(f)
	}

	e := a + 128
	heatpumpState = byte(e)
	fmt.Printf("set DHW temperature to   %d", a)
	command = []byte{0xf1, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, heatpumpState, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if config.Loghex == true {
		logHex(command, len(command))
	}
	commandsToSend[xid.New()] = command
}

func handleSetPowerfulMode(mclient mqtt.Client, msg mqtt.Message) {
	var command []byte
	var heatpumpState byte

	a, er := strconv.Atoi(string(msg.Payload()))
	if er != nil {
		f, _ := strconv.ParseFloat(string(msg.Payload()), 64)
		a = int(f)
	}

	e := a + 73
	heatpumpState = byte(e)
	fmt.Printf("set powerful mode to  %d", a)
	command = []byte{0xf1, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, heatpumpState, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if config.Loghex == true {
		logHex(command, len(command))
	}
	commandsToSend[xid.New()] = command
}

func handleSetHolidayMode(mclient mqtt.Client, msg mqtt.Message) {
	var command []byte
	var heatpumpState byte
	e := 16
	a, er := strconv.Atoi(string(msg.Payload()))
	if er != nil {
		f, _ := strconv.ParseFloat(string(msg.Payload()), 64)
		a = int(f)
	}

	if a == 1 {
		e = 32
	}
	heatpumpState = byte(e)
	fmt.Printf("set holiday mode to  %d", heatpumpState)
	command = []byte{0xf1, 0x6c, 0x01, 0x10, 0x00, heatpumpState, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if config.Loghex == true {
		logHex(command, len(command))
	}
	commandsToSend[xid.New()] = command
}

func handleSetForceSterilization(mclient mqtt.Client, msg mqtt.Message) {
	var command []byte
	var heatpumpState byte
	e := 0
	a, er := strconv.Atoi(string(msg.Payload()))
	if er != nil {
		f, _ := strconv.ParseFloat(string(msg.Payload()), 64)
		a = int(f)
	}

	if a == 1 {
		e = 4
	}
	heatpumpState = byte(e)
	fmt.Printf("set force sterilization  mode to %d", heatpumpState)
	command = []byte{0xf1, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, 0x00, heatpumpState, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if config.Loghex == true {
		logHex(command, len(command))
	}
	commandsToSend[xid.New()] = command
}

func handleSetForceDefrost(mclient mqtt.Client, msg mqtt.Message) {
	var command []byte
	var heatpumpState byte
	e := 0
	a, er := strconv.Atoi(string(msg.Payload()))
	if er != nil {
		f, _ := strconv.ParseFloat(string(msg.Payload()), 64)
		a = int(f)
	}

	if a == 1 {
		e = 2
	}
	heatpumpState = byte(e)
	fmt.Printf("set force defrost mode to %d", heatpumpState)
	command = []byte{0xf1, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, 0x00, heatpumpState, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if config.Loghex == true {
		logHex(command, len(command))
	}
	commandsToSend[xid.New()] = command
}

func handleSetForceDHW(mclient mqtt.Client, msg mqtt.Message) {
	var command []byte
	var heatpumpState byte
	e := 64
	a, er := strconv.Atoi(string(msg.Payload()))
	if er != nil {
		f, _ := strconv.ParseFloat(string(msg.Payload()), 64)
		a = int(f)
	}

	if a == 1 {
		e = 128
	}
	heatpumpState = byte(e)
	fmt.Printf("set force DHW mode to %d", heatpumpState)
	command = []byte{0xf1, 0x6c, 0x01, 0x10, heatpumpState, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if config.Loghex == true {
		logHex(command, len(command))
	}
	commandsToSend[xid.New()] = command
}

func handleSetZ1HeatRequestTemperature(mclient mqtt.Client, msg mqtt.Message) {
	var command []byte
	var requestTemp byte
	e, er := strconv.Atoi(string(msg.Payload()))
	if er != nil {
		f, _ := strconv.ParseFloat(string(msg.Payload()), 64)
		e = int(f)
	}

	e = e + 128
	requestTemp = byte(e)
	fmt.Printf("set z1 heat request temperature to %d", requestTemp-128)
	command = []byte{0xf1, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, requestTemp, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if config.Loghex == true {
		logHex(command, len(command))
	}
	commandsToSend[xid.New()] = command
}

func handleSetZ1CoolRequestTemperature(mclient mqtt.Client, msg mqtt.Message) {
	var command []byte
	var requestTemp byte
	e, er := strconv.Atoi(string(msg.Payload()))
	if er != nil {
		f, _ := strconv.ParseFloat(string(msg.Payload()), 64)
		e = int(f)
	}
	e = e + 128
	requestTemp = byte(e)
	fmt.Printf("set z1 cool request temperature to %d", requestTemp-128)
	command = []byte{0xf1, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, requestTemp, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if config.Loghex == true {
		logHex(command, len(command))
	}
	commandsToSend[xid.New()] = command
}

func handleSetZ2HeatRequestTemperature(mclient mqtt.Client, msg mqtt.Message) {
	var command []byte
	var requestTemp byte
	e, er := strconv.Atoi(string(msg.Payload()))
	if er != nil {
		f, _ := strconv.ParseFloat(string(msg.Payload()), 64)
		e = int(f)
	}
	e = e + 128
	requestTemp = byte(e)
	fmt.Printf("set z2 heat request temperature to %d", requestTemp-128)
	command = []byte{0xf1, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, requestTemp, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if config.Loghex == true {
		logHex(command, len(command))
	}
	commandsToSend[xid.New()] = command
}

func handleSetZ2CoolRequestTemperature(mclient mqtt.Client, msg mqtt.Message) {
	var command []byte
	var requestTemp byte
	e, er := strconv.Atoi(string(msg.Payload()))
	if er != nil {
		f, _ := strconv.ParseFloat(string(msg.Payload()), 64)
		e = int(f)
	}
	e = e + 128
	requestTemp = byte(e)
	fmt.Printf("set z2 cool request temperature to %d", requestTemp-128)
	command = []byte{0xf1, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, requestTemp, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if config.Loghex == true {
		logHex(command, len(command))
	}
	commandsToSend[xid.New()] = command
}

func handleSetQuietMode(mclient mqtt.Client, msg mqtt.Message) {
	var command []byte
	var quietMode byte

	e, er := strconv.Atoi(string(msg.Payload()))
	if er != nil {
		f, _ := strconv.ParseFloat(string(msg.Payload()), 64)
		e = int(f)
	}
	e = (e + 1) * 8

	quietMode = byte(e)
	fmt.Printf("set Quiet mode to %d", quietMode/8-1)
	command = []byte{0xf1, 0x6c, 0x01, 0x10, 0x00, 0x00, 0x00, quietMode, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if config.Loghex == true {
		logHex(command, len(command))
	}
	commandsToSend[xid.New()] = command
}

func handleSetHeatpump(mclient mqtt.Client, msg mqtt.Message) {
	var command []byte
	var heatpumpState byte

	e := 1
	a, er := strconv.Atoi(string(msg.Payload()))
	if er != nil {
		f, _ := strconv.ParseFloat(string(msg.Payload()), 64)
		a = int(f)
	}
	if a == 1 {
		e = 2
	}

	heatpumpState = byte(e)
	fmt.Printf("set heatpump state to %d", heatpumpState)
	command = []byte{0xf1, 0x6c, 0x01, 0x10, heatpumpState, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	if config.Loghex == true {
		logHex(command, len(command))
	}
	commandsToSend[xid.New()] = command
}

func logMessage(a string) {
	fmt.Println(a)
}

func logHex(command []byte, length int) {
	fmt.Printf("% X \n", string(command))

}

func calcChecksum(command []byte, length int) byte {
	var chk byte
	chk = 0
	for i := 0; i < length; i++ {
		chk += command[i]
	}
	chk = (chk ^ 0xFF) + 01
	return chk
}

func parseTopicList2() {

	// Loop through lines & turn into object
	for key := range allTopics {
		var data topicData
		if _, err := toml.DecodeFile(configfile, &data); err != nil {
			log.Fatal(err)
		}
		allTopics[key] = data
		//a	fmt.Println(data)
		//EncodeTopicsToTOML(TNUM, data)

	}
}

func sendCommand(command []byte, length int) bool {

	var chk byte
	chk = calcChecksum(command, length)
	var bytesSent int

	bytesSent, err := serialPort.Write(command) //first send command
	_, err = serialPort.Write([]byte{chk})      //then calculcated checksum byte afterwards
	if err != nil {
		fmt.Println(err)
	}
	logMsg := fmt.Sprintf("sent bytes: %d with checksum: %d ", bytesSent, int(chk))
	logMessage(logMsg)

	if config.Loghex == true {
		logHex(command, length)
	}
	//readSerial()
	//allowreadtime = millis() + SERIALTIMEOUT //set allowreadtime when to timeout the answer of this command
	return true
}

// func pushCommandBuffer(command []byte , length int) {
// 	if (commandsInBuffer < MAXCOMMANDSINBUFFER) {
// 	  command_struct* newCommand = new command_struct;
// 	  newCommand->length = length;
// 	  for (int i = 0 ; i < length ; i++) {
// 		newCommand->value[i] = command[i];
// 	  }
// 	  newCommand->next = commandBuffer;
// 	  commandBuffer = newCommand;
// 	  commandsInBuffer++;
// 	}
// 	else {
// 	  log_message("Too much commands already in buffer. Ignoring this commands.");
// 	}
//   }

func readSerial(MC mqtt.Client, MT mqtt.Token) bool {

	dataLength := 203

	totalreads++
	data := make([]byte, dataLength)
	n, err := serialPort.Read(data)
	if err != nil {
		log.Fatal(err)
	}
	if n == 0 {
		fmt.Println("\nEOF")

	}

	//panasonic read is always 203 on valid receive, if not yet there wait for next read
	logMessage("Received 203 bytes data\n")
	if config.Loghex {
		logHex(data, dataLength)
	}
	if !isValidReceiveHeader(data) {
		logMessage("Received wrong header!\n")
		dataLength = 0 //for next attempt;
		return false
	}
	if !isValidReceiveChecksum(data) {
		logMessage("Checksum received false!")
		dataLength = 0 //for next attempt
		return false
	}
	logMessage("Checksum and header received ok!")
	dataLength = 0 //for next attempt
	goodreads++
	readpercentage = ((goodreads / totalreads) * 100)
	logMsg := fmt.Sprintf("Total reads : %f and total good reads : %f (%.2f %%)", totalreads, goodreads, readpercentage)
	logMessage(logMsg)
	decodeHeatpumpData(data, MC, MT)
	token := MC.Publish(fmt.Sprintf("%s/LWT", config.MqttSetBase), byte(0), false, "Online")
	if token.Wait() && token.Error() != nil {
		fmt.Printf("Fail to publish, %v", token.Error())
	}
	return true

}

func isValidReceiveHeader(data []byte) bool {
	return ((data[0] == 0x71) && (data[1] == 0xC8) && (data[2] == 0x01) && (data[3] == 0x10))
}

func isValidReceiveChecksum(data []byte) bool {
	var chk byte
	chk = 0
	for i := 0; i < len(data); i++ {
		chk += data[i]
	}
	return (chk == 0) //all received bytes + checksum should result in 0
}

func callTopicFunction(data byte, f func(data byte) string) string {
	return f(data)
}

func getBit7and8(input byte) string {
	return fmt.Sprintf("%d", (input&0b11)-1)
}

func getBit3and4and5(input byte) string {
	return fmt.Sprintf("%d", ((input>>3)&0b111)-1)
}

func getIntMinus1Times10(input byte) string {
	value := int(input) - 1
	return fmt.Sprintf("%d", value*10)

}

func getIntMinus1Times50(input byte) string {
	value := int(input) - 1
	return fmt.Sprintf("%d", value*50)

}

func unknown(input byte) string {
	return "-1"
}

func getIntMinus128(input byte) string {
	value := int(input) - 128
	return fmt.Sprintf("%d", value)
}

func getIntMinus1Div5(input byte) string {
	value := int(input) - 1
	var out float32
	out = float32(value) / 5
	return fmt.Sprintf("%.2f", out)

}

func getRight3bits(input byte) string {
	return fmt.Sprintf("%d", (input&0b111)-1)

}

func getBit1and2(input byte) string {
	return fmt.Sprintf("%d", (input>>6)-1)

}

func getOpMode(input byte) string {
	switch int(input) {
	case 82:
		return "0"
	case 83:
		return "1"
	case 89:
		return "2"
	case 97:
		return "3"
	case 98:
		return "4"
	case 99:
		return "5"
	case 105:
		return "6"
	case 90:
		return "7"
	case 106:
		return "8"
	default:
		return "-1"
	}
}

func getIntMinus1(input byte) string {
	value := int(input) - 1
	return fmt.Sprintf("%d", value)
}

func getEnergy(input byte) string {
	value := (int(input) - 1) * 200
	return fmt.Sprintf("%d", value)
}

func getBit3and4(input byte) string {
	return fmt.Sprintf("%d", ((input>>4)&0b11)-1)

}

func getBit5and6(input byte) string {
	return fmt.Sprintf("%d", ((input>>2)&0b11)-1)

}

func getPumpFlow(data []byte) string { // TOP1 //
	PumpFlow1 := int(data[170])
	PumpFlow2 := ((float64(data[169]) - 1) / 256)
	PumpFlow := float64(PumpFlow1) + PumpFlow2
	//return String(PumpFlow,2);
	return fmt.Sprintf("%.2f", PumpFlow)
}

func getErrorInfo(data []byte) string { // TOP44 //
	errorType := int(data[113])
	errorNumber := int(data[114]) - 17
	var errorString string
	switch errorType {
	case 177: //B1=F type error
		errorString = fmt.Sprintf("F%02X", errorNumber)

	case 161: //A1=H type error
		errorString = fmt.Sprintf("H%02X", errorNumber)

	default:
		errorString = fmt.Sprintf("No error")

	}
	return errorString
}

func decodeHeatpumpData(data []byte, mclient mqtt.Client, token mqtt.Token) {

	var updatenow bool = false
	m := map[string]func(byte) string{
		"getBit7and8":         getBit7and8,
		"unknown":             unknown,
		"getRight3bits":       getRight3bits,
		"getIntMinus1Div5":    getIntMinus1Div5,
		"getIntMinus1Times50": getIntMinus1Times50,
		"getIntMinus1Times10": getIntMinus1Times10,
		"getBit3and4and5":     getBit3and4and5,
		"getIntMinus128":      getIntMinus128,
		"getBit1and2":         getBit1and2,
		"getOpMode":           getOpMode,
		"getIntMinus1":        getIntMinus1,
		"getEnergy":           getEnergy,
		"getBit5and6":         getBit5and6,

		"getBit3and4": getBit3and4,
	}

	// 	if (millis() > nextalldatatime) {
	// 	  updatenow = true;
	// 	  nextalldatatime = millis() + UPDATEALLTIME;
	// 	}
	for k, v := range allTopics {
		var inputByte byte
		var topicValue string
		var value string
		switch k {
		case 1:
			topicValue = getPumpFlow(data)
		case 11:
			d := make([]byte, 2)
			d[0] = data[183]
			d[1] = data[182]
			topicValue = fmt.Sprintf("%d", int(binary.BigEndian.Uint16(d))-1)
		case 12:
			d := make([]byte, 2)
			d[0] = data[180]
			d[1] = data[179]
			topicValue = fmt.Sprintf("%d", int(binary.BigEndian.Uint16(d))-1)
		case 90:
			d := make([]byte, 2)
			d[0] = data[186]
			d[1] = data[185]
			topicValue = fmt.Sprintf("%d", int(binary.BigEndian.Uint16(d))-1)
		case 91:
			d := make([]byte, 2)
			d[0] = data[189]
			d[1] = data[188]
			topicValue = fmt.Sprintf("%d", int(binary.BigEndian.Uint16(d))-1)
		case 44:
			topicValue = getErrorInfo(data)
		default:
			inputByte = data[v.TopicBit]
			if _, ok := m[v.TopicFunction]; ok {
				topicValue = callTopicFunction(inputByte, m[v.TopicFunction])
			} else {
				fmt.Println("NIE MA FUNKCJI", v.TopicFunction)
			}

		}

		if (updatenow) || (actData[k] != topicValue) {
			actData[k] = topicValue
			fmt.Printf("received TOP%d %s: %s \n", k, v.TopicName, topicValue)
			if config.Aquarea2mqttCompatible {
				TOP := "aquarea/state/" + fmt.Sprintf("%s/%s", config.Aquarea2mqttPumpID, v.TopicA2M)
				value = strings.TrimSpace(topicValue)
				value = strings.ToUpper(topicValue)
				fmt.Println("Publikuje do ", TOP, "warosc", string(value))
				token = mclient.Publish(TOP, byte(0), false, value)
				if token.Wait() && token.Error() != nil {
					fmt.Printf("Fail to publish, %v", token.Error())
				}
			}
			TOP := fmt.Sprintf("%s/%s", config.MqttTopicBase, v.TopicName)
			fmt.Println("Publikuje do ", TOP, "warosc", string(topicValue))
			token = mclient.Publish(TOP, byte(0), false, topicValue)
			if token.Wait() && token.Error() != nil {
				fmt.Printf("Fail to publish, %v", token.Error())
			}

		}

	}

}

func parseTopicList3() {
	allTopics[0].TopicNumber = 0
	allTopics[0].TopicName = "Heatpump_State"
	allTopics[0].TopicType = "binary_sensor"
	allTopics[0].TopicBit = 4
	allTopics[0].TopicFunction = "getBit7and8"
	allTopics[0].TopicUnit = "OffOn"
	allTopics[0].TopicA2M = "RunningStatus"
	allTopics[1].TopicNumber = 1
	allTopics[1].TopicName = "Pump_Flow"
	allTopics[1].TopicBit = 0
	allTopics[1].TopicDisplayUnit = "L/m"

	allTopics[1].TopicFunction = "unknown"
	allTopics[1].TopicUnit = "LitersPerMin"
	allTopics[1].TopicA2M = "WaterFlow"
	allTopics[10].TopicNumber = 10
	allTopics[10].TopicName = "DHW_Temp"
	allTopics[10].TopicBit = 141
	allTopics[10].TopicDisplayUnit = "°C"

	allTopics[10].TopicFunction = "getIntMinus128"
	allTopics[10].TopicUnit = "Celsius"
	allTopics[10].TopicA2M = "DailyWaterTankActualTemperature"
	allTopics[11].TopicNumber = 11
	allTopics[11].TopicName = "Operations_Hours"
	allTopics[11].TopicBit = 0
	allTopics[11].TopicFunction = "unknown"
	allTopics[11].TopicDisplayUnit = "h"

	allTopics[11].TopicUnit = "Hours"
	allTopics[11].TopicA2M = ""
	allTopics[12].TopicNumber = 12
	allTopics[12].TopicName = "Operations_Counter"
	allTopics[12].TopicBit = 0
	allTopics[12].TopicFunction = "unknown"
	allTopics[12].TopicUnit = "Counter"
	allTopics[12].TopicA2M = ""
	allTopics[13].TopicNumber = 13
	allTopics[13].TopicName = "Main_Schedule_State"
	allTopics[13].TopicBit = 5
	allTopics[13].TopicType = "binary_sensor"
	allTopics[13].TopicFunction = "getBit1and2"
	allTopics[13].TopicUnit = "DisabledEnabled"
	allTopics[13].TopicA2M = ""
	allTopics[14].TopicNumber = 14
	allTopics[14].TopicName = "Outside_Temp"
	allTopics[14].TopicDisplayUnit = "°C"

	allTopics[14].TopicBit = 142
	allTopics[14].TopicFunction = "getIntMinus128"
	allTopics[14].TopicUnit = "Celsius"
	allTopics[14].TopicA2M = "OutdoorTemperature"
	allTopics[15].TopicNumber = 15
	allTopics[15].TopicName = "Heat_Energy_Production"
	allTopics[15].TopicBit = 194
	allTopics[15].TopicDisplayUnit = "W"

	allTopics[15].TopicFunction = "getEnergy"
	allTopics[15].TopicUnit = "Watt"
	allTopics[15].TopicA2M = ""
	allTopics[16].TopicNumber = 16
	allTopics[16].TopicName = "Heat_Energy_Consumption"
	allTopics[16].TopicBit = 193
	allTopics[16].TopicFunction = "getEnergy"
	allTopics[16].TopicDisplayUnit = "W"
	allTopics[16].TopicUnit = "Watt"
	allTopics[16].TopicA2M = ""
	allTopics[17].TopicNumber = 17
	allTopics[17].TopicName = "Powerful_Mode_Time"
	allTopics[17].TopicBit = 7
	allTopics[17].TopicDisplayUnit = "Min"
	allTopics[17].TopicValueTemplate = `{{ (value | int) * 30 }}`

	allTopics[17].TopicFunction = "getRight3bits"
	allTopics[17].TopicUnit = "Powerfulmode"
	allTopics[17].TopicA2M = ""
	allTopics[18].TopicNumber = 18
	allTopics[18].TopicName = "Quiet_Mode_Level"
	allTopics[18].TopicBit = 7
	allTopics[18].TopicFunction = "getBit3and4and5"
	allTopics[18].TopicUnit = "Quietmode"
	allTopics[18].TopicValueTemplate = `{%- if value == "4" -%} Scheduled {%- else -%} {{ value }} {%- endif -%}`

	allTopics[18].TopicA2M = ""
	allTopics[19].TopicNumber = 19
	allTopics[19].TopicName = "Holiday_Mode_State"
	allTopics[19].TopicBit = 5
	allTopics[19].TopicType = "binary_sensor"
	allTopics[19].TopicFunction = "getBit3and4"
	allTopics[19].TopicUnit = "HolidayState"
	allTopics[19].TopicA2M = ""
	allTopics[2].TopicNumber = 2
	allTopics[2].TopicName = "Force_DHW_State"
	allTopics[2].TopicBit = 4
	allTopics[2].TopicType = "binary_sensor"

	allTopics[2].TopicFunction = "getBit1and2"
	allTopics[2].TopicUnit = "DisabledEnabled"
	allTopics[2].TopicA2M = ""
	allTopics[20].TopicNumber = 20
	allTopics[20].TopicName = "ThreeWay_Valve_State"
	allTopics[20].TopicBit = 111
	allTopics[20].TopicFunction = "getBit7and8"
	allTopics[20].TopicValueTemplate = `{%- if value == "0" -%} Room {%- elif value == "1" -%} Tank {%- endif -%}`

	allTopics[20].TopicUnit = "Valve"
	allTopics[20].TopicA2M = ""
	allTopics[21].TopicNumber = 21
	allTopics[21].TopicName = "Outside_Pipe_Temp"
	allTopics[21].TopicBit = 158
	allTopics[21].TopicFunction = "getIntMinus128"
	allTopics[21].TopicUnit = "Celsius"
	allTopics[21].TopicDisplayUnit = "°C"

	allTopics[21].TopicA2M = ""
	allTopics[22].TopicNumber = 22
	allTopics[22].TopicName = "DHW_Heat_Delta"
	allTopics[22].TopicBit = 99
	allTopics[22].TopicFunction = "getIntMinus128"
	allTopics[22].TopicUnit = "Kelvin"
	allTopics[22].TopicDisplayUnit = "°K"

	allTopics[22].TopicA2M = ""
	allTopics[23].TopicNumber = 23
	allTopics[23].TopicName = "Heat_Delta"
	allTopics[23].TopicBit = 84
	allTopics[23].TopicFunction = "getIntMinus128"
	allTopics[23].TopicUnit = "Kelvin"
	allTopics[23].TopicDisplayUnit = "°K"

	allTopics[23].TopicA2M = ""
	allTopics[24].TopicNumber = 24
	allTopics[24].TopicName = "Cool_Delta"
	allTopics[24].TopicBit = 94
	allTopics[24].TopicFunction = "getIntMinus128"
	allTopics[24].TopicUnit = "Kelvin"
	allTopics[24].TopicA2M = ""
	allTopics[24].TopicDisplayUnit = "°K"

	allTopics[25].TopicNumber = 25
	allTopics[25].TopicName = "DHW_Holiday_Shift_Temp"
	allTopics[25].TopicBit = 44
	allTopics[25].TopicFunction = "getIntMinus128"
	allTopics[25].TopicUnit = "Kelvin"
	allTopics[25].TopicDisplayUnit = "°K"

	allTopics[25].TopicA2M = ""
	allTopics[26].TopicNumber = 26
	allTopics[26].TopicName = "Defrosting_State"
	allTopics[26].TopicType = "binary_sensor"

	allTopics[26].TopicBit = 111
	allTopics[26].TopicFunction = "getBit5and6"
	allTopics[26].TopicUnit = "DisabledEnabled"
	allTopics[26].TopicA2M = ""
	allTopics[27].TopicNumber = 27
	allTopics[27].TopicName = "Z1_Heat_Request_Temp"
	allTopics[27].TopicBit = 38
	allTopics[27].TopicDisplayUnit = "°C"

	allTopics[27].TopicFunction = "getIntMinus128"
	allTopics[27].TopicUnit = "Celsius"
	allTopics[27].TopicA2M = "Zone1SetpointTemperature"
	allTopics[28].TopicNumber = 28
	allTopics[28].TopicName = "Z1_Cool_Request_Temp"
	allTopics[28].TopicBit = 39
	allTopics[28].TopicFunction = "getIntMinus128"
	allTopics[28].TopicDisplayUnit = "°C"

	allTopics[28].TopicUnit = "Celsius"
	allTopics[28].TopicA2M = ""
	allTopics[29].TopicNumber = 29
	allTopics[29].TopicName = "Z1_Heat_Curve_Target_High_Temp"
	allTopics[29].TopicBit = 75
	allTopics[29].TopicDisplayUnit = "°C"

	allTopics[29].TopicFunction = "getIntMinus128"
	allTopics[29].TopicUnit = "Celsius"
	allTopics[29].TopicA2M = ""
	allTopics[3].TopicNumber = 3
	allTopics[3].TopicName = "Quiet_Mode_Schedule"
	allTopics[3].TopicBit = 7
	allTopics[3].TopicType = "binary_sensor"
	allTopics[3].TopicFunction = "getBit1and2"
	allTopics[3].TopicUnit = "DisabledEnabled"
	allTopics[3].TopicA2M = ""
	allTopics[30].TopicNumber = 30
	allTopics[30].TopicName = "Z1_Heat_Curve_Target_Low_Temp"
	allTopics[30].TopicBit = 76
	allTopics[30].TopicDisplayUnit = "°C"

	allTopics[30].TopicFunction = "getIntMinus128"
	allTopics[30].TopicUnit = "Celsius"
	allTopics[30].TopicA2M = ""
	allTopics[31].TopicNumber = 31
	allTopics[31].TopicName = "Z1_Heat_Curve_Outside_High_Temp"
	allTopics[31].TopicBit = 78
	allTopics[31].TopicDisplayUnit = "°C"

	allTopics[31].TopicFunction = "getIntMinus128"
	allTopics[31].TopicUnit = "Celsius"
	allTopics[31].TopicA2M = ""
	allTopics[32].TopicNumber = 32
	allTopics[32].TopicName = "Z1_Heat_Curve_Outside_Low_Temp"
	allTopics[32].TopicBit = 77
	allTopics[32].TopicDisplayUnit = "°C"

	allTopics[32].TopicFunction = "getIntMinus128"
	allTopics[32].TopicUnit = "Celsius"
	allTopics[32].TopicA2M = ""
	allTopics[33].TopicNumber = 33
	allTopics[33].TopicName = "Room_Thermostat_Temp"
	allTopics[33].TopicBit = 156
	allTopics[33].TopicDisplayUnit = "°C"

	allTopics[33].TopicFunction = "getIntMinus128"
	allTopics[33].TopicUnit = "Celsius"
	allTopics[33].TopicA2M = ""
	allTopics[34].TopicNumber = 34
	allTopics[34].TopicName = "Z2_Heat_Request_Temp"
	allTopics[34].TopicBit = 40
	allTopics[34].TopicDisplayUnit = "°C"

	allTopics[34].TopicFunction = "getIntMinus128"
	allTopics[34].TopicUnit = "Celsius"
	allTopics[34].TopicA2M = "Zone2SetpointTemperature"
	allTopics[35].TopicNumber = 35
	allTopics[35].TopicName = "Z2_Cool_Request_Temp"
	allTopics[35].TopicBit = 41
	allTopics[35].TopicDisplayUnit = "°C"

	allTopics[35].TopicFunction = "getIntMinus128"
	allTopics[35].TopicUnit = "Celsius"
	allTopics[35].TopicA2M = ""
	allTopics[36].TopicNumber = 36
	allTopics[36].TopicName = "Z1_Water_Temp"
	allTopics[36].TopicBit = 145
	allTopics[36].TopicFunction = "getIntMinus128"
	allTopics[36].TopicUnit = "Celsius"
	allTopics[36].TopicDisplayUnit = "°C"

	allTopics[36].TopicA2M = "Zone1WaterTemperature"
	allTopics[37].TopicNumber = 37
	allTopics[37].TopicName = "Z2_Water_Temp"
	allTopics[37].TopicBit = 146
	allTopics[37].TopicFunction = "getIntMinus128"
	allTopics[37].TopicUnit = "Celsius"
	allTopics[37].TopicDisplayUnit = "°C"

	allTopics[37].TopicA2M = "Zone2WaterTemperature"
	allTopics[38].TopicNumber = 38
	allTopics[38].TopicName = "Cool_Energy_Production"
	allTopics[38].TopicBit = 196
	allTopics[38].TopicDisplayUnit = "W"

	allTopics[38].TopicFunction = "getEnergy"
	allTopics[38].TopicUnit = "Watt"
	allTopics[38].TopicA2M = ""
	allTopics[39].TopicNumber = 39
	allTopics[39].TopicName = "Cool_Energy_Consumption"
	allTopics[39].TopicBit = 195
	allTopics[39].TopicDisplayUnit = "W"

	allTopics[39].TopicFunction = "getEnergy"
	allTopics[39].TopicUnit = "Watt"
	allTopics[39].TopicA2M = ""
	allTopics[4].TopicNumber = 4
	allTopics[4].TopicName = "Operating_Mode_State"
	allTopics[4].TopicBit = 6
	allTopics[4].TopicValueTemplate = `{%- if value == "0" -%} Heat {%- elif value == "1" -%} Cool {%- elif value == "2" -%} Auto(Heat) {%- elif value == "3" -%} DHW {%- elif value == "4" -%} Heat+DHW {%- elif value == "5" -%} Cool+DHW {%- elif value == "6" -%} Auto(Heat)+DHW {%- elif value == "7" -%} Auto(Cool) {%- elif value == "8" -%} Auto(Cool)+DHW {%- endif -%}`
	allTopics[4].TopicFunction = "getOpMode"
	allTopics[4].TopicUnit = "OpModeDesc"
	allTopics[4].TopicA2M = "WorkingMode"
	allTopics[40].TopicNumber = 40
	allTopics[40].TopicName = "DHW_Energy_Production"
	allTopics[40].TopicBit = 198
	allTopics[40].TopicFunction = "getEnergy"
	allTopics[40].TopicUnit = "Watt"
	allTopics[40].TopicDisplayUnit = "W"

	allTopics[40].TopicA2M = ""
	allTopics[41].TopicNumber = 41
	allTopics[41].TopicName = "DHW_Energy_Consumption"
	allTopics[41].TopicBit = 197
	allTopics[41].TopicFunction = "getEnergy"
	allTopics[41].TopicUnit = "Watt"
	allTopics[41].TopicDisplayUnit = "W"

	allTopics[41].TopicA2M = ""
	allTopics[42].TopicNumber = 42
	allTopics[42].TopicName = "Z1_Water_Target_Temp"
	allTopics[42].TopicBit = 147
	allTopics[42].TopicFunction = "getIntMinus128"
	allTopics[42].TopicUnit = "Celsius"
	allTopics[42].TopicA2M = ""
	allTopics[42].TopicDisplayUnit = "°C"

	allTopics[43].TopicNumber = 43
	allTopics[43].TopicName = "Z2_Water_Target_Temp"
	allTopics[43].TopicBit = 148
	allTopics[43].TopicFunction = "getIntMinus128"
	allTopics[43].TopicUnit = "Celsius"
	allTopics[43].TopicDisplayUnit = "°C"

	allTopics[43].TopicA2M = ""
	allTopics[44].TopicNumber = 44
	allTopics[44].TopicName = "Error"
	allTopics[44].TopicBit = 0
	allTopics[44].TopicFunction = "unknown"
	allTopics[44].TopicUnit = "ErrorState"
	allTopics[44].TopicA2M = ""
	allTopics[45].TopicNumber = 45
	allTopics[45].TopicName = "Room_Holiday_Shift_Temp"
	allTopics[45].TopicBit = 43
	allTopics[45].TopicFunction = "getIntMinus128"
	allTopics[45].TopicUnit = "Kelvin"
	allTopics[45].TopicDisplayUnit = "°K"

	allTopics[45].TopicA2M = ""
	allTopics[46].TopicNumber = 46
	allTopics[46].TopicName = "Buffer_Temp"
	allTopics[46].TopicBit = 149
	allTopics[46].TopicFunction = "getIntMinus128"
	allTopics[46].TopicUnit = "Celsius"
	allTopics[46].TopicDisplayUnit = "°C"

	allTopics[46].TopicA2M = "BufferTankTemperature"
	allTopics[47].TopicNumber = 47
	allTopics[47].TopicName = "Solar_Temp"
	allTopics[47].TopicBit = 150
	allTopics[47].TopicFunction = "getIntMinus128"
	allTopics[47].TopicUnit = "Celsius"
	allTopics[47].TopicDisplayUnit = "°C"

	allTopics[47].TopicA2M = ""
	allTopics[48].TopicNumber = 48
	allTopics[48].TopicName = "Pool_Temp"
	allTopics[48].TopicBit = 151
	allTopics[48].TopicFunction = "getIntMinus128"
	allTopics[48].TopicUnit = "Celsius"
	allTopics[48].TopicDisplayUnit = "°C"

	allTopics[48].TopicA2M = ""
	allTopics[49].TopicNumber = 49
	allTopics[49].TopicName = "Main_Hex_Outlet_Temp"
	allTopics[49].TopicBit = 154
	allTopics[49].TopicDisplayUnit = "°C"

	allTopics[49].TopicFunction = "getIntMinus128"
	allTopics[49].TopicUnit = "Celsius"
	allTopics[49].TopicA2M = ""
	allTopics[5].TopicNumber = 5
	allTopics[5].TopicName = "Main_Inlet_Temp"
	allTopics[5].TopicBit = 143
	allTopics[5].TopicFunction = "getIntMinus128"
	allTopics[5].TopicUnit = "Celsius"
	allTopics[5].TopicDisplayUnit = "°C"
	allTopics[5].TopicA2M = "WaterInleet"
	allTopics[50].TopicNumber = 50
	allTopics[50].TopicName = "Discharge_Temp"
	allTopics[50].TopicBit = 155
	allTopics[50].TopicFunction = "getIntMinus128"
	allTopics[50].TopicUnit = "Celsius"
	allTopics[50].TopicDisplayUnit = "°C"

	allTopics[50].TopicA2M = ""
	allTopics[51].TopicNumber = 51
	allTopics[51].TopicName = "Inside_Pipe_Temp"
	allTopics[51].TopicBit = 157
	allTopics[51].TopicFunction = "getIntMinus128"
	allTopics[51].TopicUnit = "Celsius"
	allTopics[51].TopicDisplayUnit = "°C"

	allTopics[51].TopicA2M = ""
	allTopics[52].TopicNumber = 52
	allTopics[52].TopicName = "Defrost_Temp"
	allTopics[52].TopicBit = 159
	allTopics[52].TopicFunction = "getIntMinus128"
	allTopics[52].TopicUnit = "Celsius"
	allTopics[52].TopicA2M = ""
	allTopics[52].TopicDisplayUnit = "°C"

	allTopics[53].TopicNumber = 53
	allTopics[53].TopicDisplayUnit = "°C"

	allTopics[53].TopicName = "Eva_Outlet_Temp"
	allTopics[53].TopicBit = 160
	allTopics[53].TopicFunction = "getIntMinus128"
	allTopics[53].TopicUnit = "Celsius"
	allTopics[53].TopicA2M = ""
	allTopics[54].TopicNumber = 54
	allTopics[54].TopicName = "Bypass_Outlet_Temp"
	allTopics[54].TopicBit = 161
	allTopics[54].TopicDisplayUnit = "°C"

	allTopics[54].TopicFunction = "getIntMinus128"
	allTopics[54].TopicUnit = "Celsius"
	allTopics[54].TopicA2M = ""
	allTopics[55].TopicNumber = 55
	allTopics[55].TopicName = "Ipm_Temp"
	allTopics[55].TopicBit = 162
	allTopics[55].TopicDisplayUnit = "°C"

	allTopics[55].TopicFunction = "getIntMinus128"
	allTopics[55].TopicUnit = "Celsius"
	allTopics[55].TopicA2M = ""
	allTopics[56].TopicNumber = 56
	allTopics[56].TopicName = "Z1_Temp"
	allTopics[56].TopicBit = 139
	allTopics[56].TopicFunction = "getIntMinus128"
	allTopics[56].TopicUnit = "Celsius"
	allTopics[56].TopicDisplayUnit = "°C"

	allTopics[56].TopicA2M = "Zone1ActualTemperature"
	allTopics[57].TopicNumber = 57
	allTopics[57].TopicName = "Z2_Temp"
	allTopics[57].TopicBit = 140
	allTopics[57].TopicFunction = "getIntMinus128"
	allTopics[57].TopicUnit = "Celsius"
	allTopics[57].TopicDisplayUnit = "°C"

	allTopics[57].TopicA2M = "Zone2ActualTemperature"
	allTopics[58].TopicNumber = 58
	allTopics[58].TopicName = "DHW_Heater_State"
	allTopics[58].TopicBit = 9
	allTopics[58].TopicType = "binary_sensor"

	allTopics[58].TopicFunction = "getBit5and6"
	allTopics[58].TopicUnit = "BlockedFree"
	allTopics[58].TopicA2M = ""
	allTopics[59].TopicNumber = 59
	allTopics[59].TopicName = "Room_Heater_State"
	allTopics[59].TopicBit = 9
	allTopics[59].TopicType = "binary_sensor"

	allTopics[59].TopicFunction = "getBit7and8"
	allTopics[59].TopicUnit = "BlockedFree"
	allTopics[59].TopicA2M = ""
	allTopics[6].TopicNumber = 6
	allTopics[6].TopicName = "Main_Outlet_Temp"
	allTopics[6].TopicBit = 144
	allTopics[6].TopicFunction = "getIntMinus128"
	allTopics[6].TopicUnit = "Celsius"
	allTopics[6].TopicDisplayUnit = "°C"

	allTopics[6].TopicA2M = "WaterOutleet"
	allTopics[60].TopicNumber = 60
	allTopics[60].TopicType = "binary_sensor"

	allTopics[60].TopicName = "Internal_Heater_State"
	allTopics[60].TopicBit = 112
	allTopics[60].TopicFunction = "getBit7and8"
	allTopics[60].TopicUnit = "InactiveActive"
	allTopics[60].TopicA2M = ""
	allTopics[61].TopicNumber = 61
	allTopics[61].TopicName = "External_Heater_State"
	allTopics[61].TopicBit = 112
	allTopics[61].TopicFunction = "getBit5and6"
	allTopics[61].TopicUnit = "InactiveActive"
	allTopics[61].TopicA2M = ""
	allTopics[61].TopicType = "binary_sensor"

	allTopics[62].TopicNumber = 62
	allTopics[62].TopicName = "Fan1_Motor_Speed"
	allTopics[62].TopicBit = 173
	allTopics[62].TopicDisplayUnit = "R/min"

	allTopics[62].TopicFunction = "getIntMinus1Times10"
	allTopics[62].TopicUnit = "RotationsPerMin"
	allTopics[62].TopicA2M = ""
	allTopics[63].TopicNumber = 63
	allTopics[63].TopicName = "Fan2_Motor_Speed"
	allTopics[63].TopicBit = 174
	allTopics[63].TopicDisplayUnit = "R/min"
	allTopics[63].TopicFunction = "getIntMinus1Times10"
	allTopics[63].TopicUnit = "RotationsPerMin"
	allTopics[63].TopicA2M = ""
	allTopics[64].TopicNumber = 64
	allTopics[64].TopicName = "High_Pressure"
	allTopics[64].TopicBit = 163
	allTopics[64].TopicDisplayUnit = "Kgf/cm2"
	allTopics[64].TopicFunction = "getIntMinus1Div5"
	allTopics[64].TopicUnit = "Pressure"
	allTopics[64].TopicA2M = ""
	allTopics[65].TopicNumber = 65
	allTopics[65].TopicDisplayUnit = "R/mini"

	allTopics[65].TopicName = "Pump_Speed"
	allTopics[65].TopicBit = 171
	allTopics[65].TopicFunction = "getIntMinus1Times50"
	allTopics[65].TopicUnit = "RotationsPerMin"
	allTopics[65].TopicA2M = "PumpSpeed"
	allTopics[66].TopicNumber = 66
	allTopics[66].TopicName = "Low_Pressure"
	allTopics[66].TopicBit = 164
	allTopics[66].TopicDisplayUnit = "Kgf/cm2"

	allTopics[66].TopicFunction = "getIntMinus1"
	allTopics[66].TopicUnit = "Pressure"
	allTopics[66].TopicA2M = ""
	allTopics[67].TopicNumber = 67
	allTopics[67].TopicName = "Compressor_Current"
	allTopics[67].TopicBit = 165
	allTopics[67].TopicDisplayUnit = "A"

	allTopics[67].TopicFunction = "getIntMinus1Div5"
	allTopics[67].TopicUnit = "Ampere"
	allTopics[67].TopicA2M = ""
	allTopics[68].TopicNumber = 68
	allTopics[68].TopicName = "Force_Heater_State"
	allTopics[68].TopicBit = 5
	allTopics[68].TopicType = "binary_sensor"
	allTopics[68].TopicFunction = "getBit5and6"
	allTopics[68].TopicUnit = "InactiveActive"
	allTopics[68].TopicA2M = ""
	allTopics[69].TopicNumber = 69
	allTopics[69].TopicName = "Sterilization_State"
	allTopics[69].TopicBit = 117
	allTopics[69].TopicType = "binary_sensor"
	allTopics[69].TopicFunction = "getBit5and6"
	allTopics[69].TopicUnit = "InactiveActive"
	allTopics[69].TopicA2M = ""
	allTopics[7].TopicNumber = 7
	allTopics[7].TopicName = "Main_Target_Temp"
	allTopics[7].TopicBit = 153
	allTopics[7].TopicFunction = "getIntMinus128"
	allTopics[7].TopicUnit = "Celsius"
	allTopics[7].TopicDisplayUnit = "°C"

	allTopics[7].TopicA2M = ""
	allTopics[70].TopicNumber = 70
	allTopics[70].TopicName = "Sterilization_Temp"
	allTopics[70].TopicBit = 100
	allTopics[70].TopicDisplayUnit = "°C"

	allTopics[70].TopicFunction = "getIntMinus128"
	allTopics[70].TopicUnit = "Celsius"
	allTopics[70].TopicA2M = ""
	allTopics[71].TopicNumber = 71
	allTopics[71].TopicName = "Sterilization_Max_Time"
	allTopics[71].TopicBit = 101
	allTopics[71].TopicFunction = "getIntMinus1"
	allTopics[71].TopicUnit = "Minutes"
	allTopics[71].TopicDisplayUnit = "min"

	allTopics[71].TopicA2M = ""
	allTopics[72].TopicNumber = 72
	allTopics[72].TopicName = "Z1_Cool_Curve_Target_High_Temp"
	allTopics[72].TopicBit = 86
	allTopics[72].TopicFunction = "getIntMinus128"
	allTopics[72].TopicUnit = "Celsius"
	allTopics[72].TopicA2M = ""
	allTopics[72].TopicDisplayUnit = "°C"

	allTopics[73].TopicNumber = 73
	allTopics[73].TopicName = "Z1_Cool_Curve_Target_Low_Temp"
	allTopics[73].TopicBit = 87
	allTopics[73].TopicFunction = "getIntMinus128"
	allTopics[73].TopicUnit = "Celsius"
	allTopics[73].TopicDisplayUnit = "°C"

	allTopics[73].TopicA2M = ""
	allTopics[74].TopicNumber = 74
	allTopics[74].TopicName = "Z1_Cool_Curve_Outside_High_Temp"
	allTopics[74].TopicBit = 88
	allTopics[74].TopicFunction = "getIntMinus128"
	allTopics[74].TopicUnit = "Celsius"
	allTopics[74].TopicDisplayUnit = "°C"

	allTopics[74].TopicA2M = ""
	allTopics[75].TopicNumber = 75
	allTopics[75].TopicName = "Z1_Cool_Curve_Outside_Low_Temp"
	allTopics[75].TopicBit = 89
	allTopics[75].TopicFunction = "getIntMinus128"
	allTopics[75].TopicUnit = "Celsius"
	allTopics[75].TopicDisplayUnit = "°C"

	allTopics[75].TopicA2M = ""
	allTopics[76].TopicNumber = 76
	allTopics[76].TopicName = "Heating_Mode"
	allTopics[76].TopicBit = 28
	allTopics[76].TopicFunction = "getBit7and8"
	allTopics[76].TopicUnit = "HeatCoolModeDesc"
	allTopics[76].TopicA2M = ""
	allTopics[77].TopicNumber = 77
	allTopics[77].TopicName = "Heating_Off_Outdoor_Temp"
	allTopics[77].TopicBit = 83
	allTopics[77].TopicFunction = "getIntMinus128"
	allTopics[77].TopicUnit = "Celsius"
	allTopics[77].TopicDisplayUnit = "°C"

	allTopics[77].TopicA2M = ""
	allTopics[78].TopicNumber = 78
	allTopics[78].TopicName = "Heater_On_Outdoor_Temp"
	allTopics[78].TopicBit = 85
	allTopics[78].TopicFunction = "getIntMinus128"
	allTopics[78].TopicUnit = "Celsius"
	allTopics[78].TopicA2M = ""
	allTopics[78].TopicDisplayUnit = "°C"

	allTopics[79].TopicNumber = 79
	allTopics[79].TopicName = "Heat_To_Cool_Temp"
	allTopics[79].TopicBit = 95
	allTopics[79].TopicFunction = "getIntMinus128"
	allTopics[79].TopicUnit = "Celsius"
	allTopics[79].TopicDisplayUnit = "°C"

	allTopics[79].TopicA2M = ""
	allTopics[8].TopicNumber = 8
	allTopics[8].TopicName = "Compressor_Freq"
	allTopics[8].TopicBit = 166
	allTopics[8].TopicFunction = "getIntMinus1"
	allTopics[8].TopicUnit = "Hertz"
	allTopics[8].TopicDisplayUnit = "hz"

	allTopics[8].TopicA2M = ""
	allTopics[80].TopicNumber = 80
	allTopics[80].TopicName = "Cool_To_Heat_Temp"
	allTopics[80].TopicBit = 96
	allTopics[80].TopicDisplayUnit = "°C"

	allTopics[80].TopicFunction = "getIntMinus128"
	allTopics[80].TopicUnit = "Celsius"
	allTopics[80].TopicA2M = ""
	allTopics[81].TopicNumber = 81
	allTopics[81].TopicName = "Cooling_Mode"
	allTopics[81].TopicBit = 28
	allTopics[81].TopicFunction = "getBit5and6"
	allTopics[81].TopicUnit = "HeatCoolModeDesc"
	allTopics[81].TopicA2M = ""
	allTopics[82].TopicNumber = 82
	allTopics[82].TopicName = "Z2_Heat_Curve_Target_High_Temp"
	allTopics[82].TopicBit = 79
	allTopics[82].TopicFunction = "getIntMinus128"
	allTopics[82].TopicUnit = "Celsius"
	allTopics[82].TopicA2M = ""
	allTopics[82].TopicDisplayUnit = "°C"

	allTopics[83].TopicNumber = 83
	allTopics[83].TopicName = "Z2_Heat_Curve_Target_Low_Temp"
	allTopics[83].TopicBit = 80
	allTopics[83].TopicFunction = "getIntMinus128"
	allTopics[83].TopicUnit = "Celsius"
	allTopics[83].TopicDisplayUnit = "°C"

	allTopics[83].TopicA2M = ""
	allTopics[84].TopicNumber = 84
	allTopics[84].TopicName = "Z2_Heat_Curve_Outside_High_Temp"
	allTopics[84].TopicBit = 81
	allTopics[84].TopicDisplayUnit = "°C"

	allTopics[84].TopicFunction = "getIntMinus128"
	allTopics[84].TopicUnit = "Celsius"
	allTopics[84].TopicA2M = ""
	allTopics[85].TopicNumber = 85
	allTopics[85].TopicName = "Z2_Heat_Curve_Outside_Low_Temp"
	allTopics[85].TopicBit = 82
	allTopics[85].TopicDisplayUnit = "°C"

	allTopics[85].TopicFunction = "getIntMinus128"
	allTopics[85].TopicUnit = "Celsius"
	allTopics[85].TopicA2M = ""
	allTopics[86].TopicNumber = 86
	allTopics[86].TopicName = "Z2_Cool_Curve_Target_High_Temp"
	allTopics[86].TopicBit = 90
	allTopics[86].TopicFunction = "getIntMinus128"
	allTopics[86].TopicUnit = "Celsius"
	allTopics[86].TopicDisplayUnit = "°C"

	allTopics[86].TopicA2M = ""
	allTopics[87].TopicNumber = 87
	allTopics[87].TopicName = "Z2_Cool_Curve_Target_Low_Temp"
	allTopics[87].TopicBit = 91
	allTopics[87].TopicDisplayUnit = "°C"

	allTopics[87].TopicFunction = "getIntMinus128"
	allTopics[87].TopicUnit = "Celsius"
	allTopics[87].TopicA2M = ""
	allTopics[88].TopicNumber = 88
	allTopics[88].TopicName = "Z2_Cool_Curve_Outside_High_Temp"
	allTopics[88].TopicBit = 92
	allTopics[88].TopicDisplayUnit = "°C"

	allTopics[88].TopicFunction = "getIntMinus128"
	allTopics[88].TopicUnit = "Celsius"
	allTopics[88].TopicA2M = ""
	allTopics[89].TopicNumber = 89
	allTopics[89].TopicName = "Z2_Cool_Curve_Outside_Low_Temp"
	allTopics[89].TopicBit = 93
	allTopics[89].TopicFunction = "getIntMinus128"
	allTopics[89].TopicUnit = "Celsius"
	allTopics[89].TopicDisplayUnit = "°C"

	allTopics[89].TopicA2M = ""
	allTopics[9].TopicNumber = 9
	allTopics[9].TopicName = "DHW_Target_Temp"
	allTopics[9].TopicBit = 42
	allTopics[9].TopicFunction = "getIntMinus128"
	allTopics[9].TopicUnit = "Celsius"
	allTopics[9].TopicDisplayUnit = "°C"

	allTopics[9].TopicA2M = "DailyWaterTankSetpointTemperature"
	allTopics[90].TopicNumber = 90
	allTopics[90].TopicName = "Room_Heater_Operations_Hours"
	allTopics[90].TopicBit = 0
	allTopics[90].TopicDisplayUnit = "h"
	allTopics[90].TopicFunction = "unknown"
	allTopics[90].TopicUnit = "Hours"
	allTopics[90].TopicA2M = ""
	allTopics[91].TopicNumber = 91
	allTopics[91].TopicName = "DHW_Heater_Operations_Hours"
	allTopics[91].TopicBit = 0
	allTopics[91].TopicDisplayUnit = "h"
	allTopics[91].TopicFunction = "unknown"
	allTopics[91].TopicUnit = "Hours"
	allTopics[91].TopicA2M = ""

	allTopics[92].TopicNumber = 92
	allTopics[92].TopicName = "Heat_Pump_Model"
	allTopics[92].TopicBit = 132
	allTopics[92].TopicDisplayUnit = "Model"
	allTopics[92].TopicFunction = "unknown"
	allTopics[92].TopicUnit = "Model"
	allTopics[92].TopicA2M = ""

	allTopics[93].TopicNumber = 93
	allTopics[93].TopicName = "Pump_Duty"
	allTopics[93].TopicBit = 172
	allTopics[93].TopicDisplayUnit = "Duty"
	allTopics[93].TopicFunction = "getIntMinus1"
	allTopics[93].TopicUnit = "Duty"
	allTopics[93].TopicA2M = ""

	allTopics[94].TopicNumber = 94
	allTopics[94].TopicName = "Zones_State"
	allTopics[94].TopicBit = 6
	allTopics[94].TopicDisplayUnit = "ZonesState"
	allTopics[94].TopicFunction = "getBit1and2"
	allTopics[94].TopicUnit = "ZonesState"
	allTopics[94].TopicA2M = ""

}
