#!/bin/bash

configure_app() {
  echo "Enter the GOOGLE_CLIENT_ID:"
  read -s GOOGLE_CLIENT_ID
  echo "Enter the GOOGLE_CLIENT_SECRET:"
  read -s GOOGLE_CLIENT_SECRET
  echo "Enter the REDIRECT_URL:"
  read -s REDIRECT_URL
  echo "Enter the AUTHORIZED_USERS as a comma separated list:"
  read -s AUTHORIZED_USERS
  echo "Enter the SESSION_SECRET:"
  read -s SESSION_SECRET

  # / as a delimter fails on REDIRECT_URL, using ~ instead
  sed "s~{{.GOOGLE_CLIENT_ID}}~${GOOGLE_CLIENT_ID}~" ${archive_path}/install/pi-alarm.service.tmpl \
       | sed "s~{{.GOOGLE_CLIENT_SECRET}}~${GOOGLE_CLIENT_SECRET}~" \
       | sed "s~{{.REDIRECT_URL}}~${REDIRECT_URL}~" \
       | sed "s~{{.AUTHORIZED_USERS}}~${AUTHORIZED_USERS}~" \
       | sed "s~{{.SESSION_SECRET}}~${SESSION_SECRET}~" > ${archive_path}/install/pi-alarm.service

  sudo mv ${archive_path}/install/pi-alarm.service /etc/systemd/system/
  rm ${archive_path}/install/pi-alarm.service.tmpl
}

install_pagekite() {
  curl -s -O https://pagekite.net/pk/pagekite.py
  chmod +x pagekite.py
  sudo mv pagekite.py /usr/local/bin/pagekite.py
  echo "Enter the pagekite name:"
  read PAGE_KITE
  sed "s/{{.PAGE_KITE}}/${PAGE_KITE}/" ${archive_path}/install/pagekite.service.tmpl > ${archive_path}/install/pagekite.service
  sudo mv ${archive_path}/install/pagekite.service /etc/systemd/system/
  echo "Entering configuration for Pagekite. Use ctrl-c after configuration is complete."
  pagekite.py 8080 ${PAGE_KITE}
}

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
configure_app

install_pagekite

sudo systemctl enable pi-alarm.service
sudo systemctl start pi-alarm.service
sudo systemctl enable pagekite.service
sudo systemctl start pagekite.service

rm -rf ${archive_path}
echo "Installation complete"
