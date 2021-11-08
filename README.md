
This project is to modify Panasonic CZ-TAW1 Firmware to send data from heat pump to MQTT instead to Aquarea Cloud (there is some POC work proving there is a posiblity to send data concurently to Aquarea Cloud and MQTT host using only modified CZ-TAW1 ,but it's not yet implemented in this project )

### This Project Contains:

- Main software (called GoHeishaMon) responsible for parsing data from Heat Pump - it's golang implementation of project https://github.com/Egyras/HeishaMon 
All MQTT topics are compatible with HeishaMon project: https://github.com/Egyras/HeishaMon/blob/master/MQTT-Topics.md
and there are two aditional topics to run command's in system runing the software but it need's another manual.

GoHeishaMon can be used without the CZ-TAW1 module on every platform supported by golang (RaspberyPi, Windows, Linux, OpenWrt routers for example) after connecting it to Heat Pump over rs232-ttl interface.
If you need help with this project you can try Slack of Heishamon project there is some people who manage this one :)

- OpenWRT Image with preinstalled GoHeishaMon (and removed A2Wmain due to copyright issues) 

CZ-TAW1 flash memory is divided for two parts called "sides". During Smart Cloud update A2Wmain software programing other side then actually it boots ,and change the side just before reboot. In this way, normally in CZ-TAW1 there are two versions of firmware: actual and previous.
Updating firmware with GoHeishaMon we use one side , and we can very easly change the side to Smsrt Cloud (A2Wmain software) by pressing all three buttons on CZ-TAW1 when GoHeishaMon works ( middle LED will change the color to RED and shortly after this it reboots to orginal SmartCloud.
Unfortunatly from Smart Cloud software changing the side without having acces to ssh console is possible only when updating other side was take place succesfully.

Summary: 

Even the GoHeishaMon is on other side you can't just change the site in orginal software to GoHeishaMon without acces to console. You have to install GoHeishaMon again. 

## Installation

This is going to change failry soon:
![image](https://user-images.githubusercontent.com/1310239/140833286-01c950ae-a072-4c7f-8952-7a1e2b648e8e.png)
