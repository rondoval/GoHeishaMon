package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
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
var config configStruct
var sending bool
var serialPort serial.Port
var err error
var goodreads float64
var totalreads float64
var readpercentage float64
var switchTopics map[string]autoDiscoverStruct

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
