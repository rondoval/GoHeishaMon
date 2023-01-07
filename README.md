# Custom firmware for the Panasonic CZ-TAW1
This is an alternative firmware for the Panasonic CZ-TAW1, an IoT adapter for the H-series heat pumps.
It consists of:

* a gateway written in Go that translates serial comms with the heat pump on the CN-CNT link to MQTT
* a port of OpenWRT (21.02.1 currently) for the CZ-TAW1

## About

### The gateway
The gateway (called GoHeishaMon or heishamon) is responsible for parsing the data received from the Heat Pump and posting it to MQTT topics. It is a reimplementation of the <https://github.com/Egyras/HeishaMon> project in Go.
**Features:**

* posting Heat Pump data to MQTT
* changing settings on the Heat Pump
* supports Home Assistant's MQTT discovery
* emulation of the Optional PCB (not tested at all)
 

GoHeishaMon can be used without the CZ-TAW1 module on a platform supported by Go. It requires a serial port connection to the Heat Pump.
The new version is running as a daemon. As a consequence, the logs are no longer written to stdout, they end up in Syslog (and MQTT topic).

*Note*
* the binary is /usr/bin/heishamon by default
* the configuration is stored in /etc/heishamon/ and is preserved on upgrades
* the service name is **heishamon**

### OpenWRT 21.02.1 image for the CZ-TAW1

**Features**

* stock OpenWRT, with up-to-date kernel (5.4.158)
* GoHeishaMon is preinstalled and running as a system service (named heishamon) 
* sysupgrade and upgrades from LuCI are working
* The CZ-TAW1's WiFi can be used as an Access Point
* ca. 6.6 MiB free on JFFS
* WPA3, LuCI over HTTPS etc.

This is pretty much stock OpenWRT, by default configured as an Acess Point/range extender, i.e. on first boot the Ethernet interface is configured as an DHCP client, and the AP is disabled. Both are in the LAN firewall zone, though one can of course reconfigure it and use the device as a router.
There is no default password. The default host name is "aquarea".

*Note*
This firmware is **not** using the factory MTD layout, i.e. the two "sides" are gone. In exchange you have a lot more space on the JFFS partiion. Of the stock partitions, u-boot, u-boot env and art are preserved. There is no easy way to migrate from stock and there is no easy way back. Back up everything.

## Installation
Beware: This is a dangerous process that may brick your device! You do it on your own responsibility.

### Prerequisites

* Serial port connection to the device
* Backup of the MTD layout, all MTD partitions and U-Boot environment
* TFTP server

Overview of the process:

* Backup MTD layout (cat /proc/mtd)
* Backup U-Boot env (fw_printenv)
* Backup **all** partitions (dd if=/dev/mtdx of=/tmp/mtdx_backup, then scp somewhere safe)
* Reboot to U-Boot
* Change boot address and options
  * *skip this if you intend to only boot it from RAM without altering the MTD*
  * setenv bootargs console=ttyS0,115200
  * setenv bootcmd bootm 0x9f050000
  * saveenv
* Download the initramfs image to RAM using TFTP (you may want to change the IP addresses in serverip and ipaddr variables)
  * tftp 0x81000000 openwrt-ath79-generic-panasonic_cz-taw1-initramfs-kernel.bin
* Boot the image
  * bootm 0x81000000
* If you want this permanet, **ssh** to OpenWRT and:
  * download openwrt-ath79-generic-panasonic_cz-taw1-squashfs-sysupgrade.bin to /tmp
  * sysupgrade /tmp/openwrt-ath79-generic-panasonic_cz-taw1-squashfs-sysupgrade.bin
  * ... or just do it from LuCI

In order to configure GoHeishaMon:

* service heishamon stop
* Edit the config file (/etc/heishamon/config.yaml)
* service heishamon start

Logs - either logread or Status/System Log in LuCI
