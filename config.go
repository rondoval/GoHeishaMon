package main

import (
	"bytes"
	"crypto/md5"
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
	Device            string `yaml:"device"`
	Loghex            bool   `yaml:"loghex"`
	ReadInterval      int    `yaml:"readInterval"`
	MqttTopicBase     string `yaml:"mqtt_topic_base"`
	MqttSetBase       string `yaml:"mqtt_set_base"`
	MqttServer        string `yaml:"mqttServer"`
	MqttPort          string `yaml:"mqttPort"`
	MqttLogin         string `yaml:"mqttLogin"`
	MqttPass          string `yaml:"mqttPass"`
	MqttClientID      string `yaml:"mqttClientID"`
	MqttKeepalive     int    `yaml:"mqttKeepalive"`
	EnableCommand     bool   `yaml:"enableCommand"`
	SleepAfterCommand int    `yaml:"sleepAfterCommand"`
	HAAutoDiscover    bool   `yaml:"haAutoDiscover"`
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
		// it's either it reboots or we can't continue
		for {
			time.Sleep(10 * time.Second)
		}
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

func updateConfig() {
	var configfile = getConfigFile()
	log.Printf("Attempting to update config file: %s", configfile)
	out, err := exec.Command("/usr/bin/usb_mount.sh").Output()
	defer exec.Command("/usr/bin/usb_umount.sh").Output()
	if err != nil {
		log.Println(err.Error())
	}
	log.Println(out)

	_, err = os.Stat("/mnt/usb/GoHeishaMonConfig.new")
	if err != nil {
		return
	}
	if !bytes.Equal(getFileChecksum(configfile), getFileChecksum("/mnt/usb/GoHeishaMonConfig.new")) {
		log.Println("Updated configuration detected on USB media... will reboot")

		_, err = exec.Command("/bin/cp", "/mnt/usb/GoHeishaMonConfig.new", configfile).Output()
		if err != nil {
			log.Printf("Can't update config file %s", configfile)
			return
		}
		_, _ = exec.Command("sync").Output()
		_, _ = exec.Command("/usr/bin/usb_umount.sh").Output()
		_, _ = exec.Command("reboot").Output()
	}
}

func getFileChecksum(f string) []byte {
	input := strings.NewReader(f)

	hash := md5.New()
	if _, err := io.Copy(hash, input); err != nil {
		log.Fatal(err)
	}
	return hash.Sum(nil)
}

func updateConfigLoop() {
	for {
		updateConfig()
		time.Sleep(time.Minute * 5)
	}
}
