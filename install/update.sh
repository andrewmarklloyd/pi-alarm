#!/bin/bash

archive_path="/tmp/pi-alarm"
mkdir -p ${archive_path}

latestVersion=$(curl --silent "https://api.github.com/repos/andrewmarklloyd/pi-alarm/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
curl -sL https://github.com/andrewmarklloyd/pi-alarm/archive/${latestVersion}.tar.gz | tar xvfz - -C "${archive_path}" --strip 1 > /dev/null

binaryUrl=$(curl -s https://api.github.com/repos/andrewmarklloyd/pi-alarm/releases/latest | jq -r '.assets[] | select(.name == "pi-alarm") | .browser_download_url')
curl -sL $binaryUrl -o ${archive_path}/pi-alarm
chmod +x ${archive_path}/pi-alarm
rm -f ./install/*
rm -f ./static/*
cp ${archive_path}/install/* install/
cp ${archive_path}/static/* static/
echo -n ${latestVersion} > /home/pi/static/version
echo -n ${latestVersion} > /home/pi/static/latestVersion
mv ${archive_path}/pi-alarm ./
rm -rf ${archive_path}
sudo systemctl restart pi-alarm.service
