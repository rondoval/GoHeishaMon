package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

type configStruct struct {
	Loghex                 bool   `yaml:"loghex"`
	Device                 string `yaml:"device"`
	ReadInterval           int    `yaml:"readInterval"`
	MqttServer             string `yaml:"mqttServer"`
	MqttPort               string `yaml:"mqttPort"`
	MqttLogin              string `yaml:"mqttLogin"`
	Aquarea2mqttCompatible bool   `yaml:"aquarea2mqttCompatible"`
	MqttTopicBase          string `yaml:"mqtt_topic_base"`
	MqttSetBase            string `yaml:"mqtt_set_base"`
	Aquarea2mqttPumpID     string `yaml:"aquarea2mqttPumpID"`
	MqttPass               string `yaml:"mqttPass"`
	MqttClientID           string `yaml:"mqttClientID"`
	MqttKeepalive          int    `yaml:"mqttKeepalive"`
	ForceRefreshTime       int    `yaml:"forceRefreshTime"`
	EnableCommand          bool   `yaml:"enableCommand"`
	SleepAfterCommand      int    `yaml:"sleepAfterCommand"`
	HAAutoDiscover         bool   `yaml:"haAutoDiscover"`
}

func getConfigFile() string {
	if runtime.GOOS != "windows" {
		return "/etc/gh/config.yaml"
	}
	return "config.yaml"
}

func readConfig() configStruct {
	var configFile = getConfigFile()

	_, err := os.Stat(configFile)
	if err != nil {
		log.Printf("Config file is missing: %s ", configFile)
		updateConfig()
	}

	_, err = os.Stat(configFile)
	if err != nil {
		log.Fatal("Config file is missing: ", configFile)
	}

	var config configStruct

	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		log.Fatal(err)
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatal(err)
	}
	return config
}

func updateConfig() bool {
	var configfile = getConfigFile()
	log.Printf("try to update configfile: %s", configfile)
	out, err := exec.Command("/usr/bin/usb_mount.sh").Output()
	if err != nil {
		log.Println(err.Error())
	}
	log.Println(out)
	_, err = os.Stat("/mnt/usb/GoHeishaMonConfig.new")
	if err != nil {
		_, _ = exec.Command("/usr/bin/usb_umount.sh").Output()
		return false
	}
	if getFileChecksum(configfile) != getFileChecksum("/mnt/usb/GoHeishaMonConfig.new") {
		log.Printf("checksum of configfile and new configfile diffrent: %s ", configfile)

		_, _ = exec.Command("/bin/cp", "/mnt/usb/GoHeishaMonConfig.new", configfile).Output()
		if err != nil {
			log.Printf("can't update configfile %s", configfile)
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

func getFileChecksum(f string) string {
	input := strings.NewReader(f)

	hash := md5.New()
	if _, err := io.Copy(hash, input); err != nil {
		log.Fatal(err)
	}
	sum := hash.Sum(nil)

	return fmt.Sprintf("%x\n", sum)

}

func updateConfigLoop() {
	for {
		updateConfig()
		time.Sleep(time.Minute * 5)

	}
}
