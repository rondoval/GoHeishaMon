package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

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
