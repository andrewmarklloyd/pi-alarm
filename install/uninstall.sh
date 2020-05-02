#!/bin/bash

sudo systemctl stop pi-alarm.service
sudo systemctl disable pi-alarm.service
sudo rm -f /etc/systemd/system/pi-alarm.service

sudo systemctl stop pagekite.service
sudo systemctl disable pagekite.service
sudo rm -f /etc/systemd/system/pagekite.service

rm -rf install/
rm pi-alarm
rm -rf private/
rm -rf public/
