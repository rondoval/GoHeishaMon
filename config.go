package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

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
