#!/bin/bash

sudo apt-get update
sudo apt-get install jq -y

archive_path="/tmp/pi-alarm"
install_dir="/home/pi"
mkdir -p ${archive_path}

latestVersion=$(curl --silent "https://api.github.com/repos/andrewmarklloyd/pi-alarm/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
curl -sL https://github.com/andrewmarklloyd/pi-alarm/archive/${latestVersion}.tar.gz | tar xvfz - -C "${archive_path}" --strip 1 > /dev/null

binaryUrl=$(curl -s https://api.github.com/repos/andrewmarklloyd/pi-alarm/releases/latest | jq -r '.assets[] | select(.name == "pi-alarm") | .browser_download_url')
curl -sL $binaryUrl -o ${archive_path}/pi-alarm
chmod +x ${archive_path}/pi-alarm
rm -f ${install_dir}/install/*
rm -f ${install_dir}/public/*
rm -f ${install_dir}/private/*

mkdir -p ${install_dir}/install/
mkdir -p ${install_dir}/public/
mkdir -p ${install_dir}/private/
cp ${archive_path}/install/* ${install_dir}/install/
cp ${archive_path}/public/* ${install_dir}/public/
cp ${archive_path}/private/* ${install_dir}/private/

echo -n ${latestVersion} > ${install_dir}/public/version
echo -n ${latestVersion} > ${install_dir}/public/latestVersion
mv ${archive_path}/pi-alarm ${install_dir}/
sudo mv ${archive_path}/install/pi-alarm.service /etc/systemd/system/
rm -rf ${archive_path}

sudo systemctl enable pi-alarm.service
sudo systemctl start pi-alarm.service
