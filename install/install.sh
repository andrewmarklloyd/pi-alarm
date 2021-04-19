#!/bin/bash

install_pagekite() {
  curl -s -O https://pagekite.net/pk/pagekite.py
  chmod +x pagekite.py
  sudo mv pagekite.py /usr/local/bin/pagekite.py
  echo "Enter the pagekite name, for example 'my-kite.pagekite.me"
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

# Configure Heroku to get secrets
echo "Enter the Heroku app name that contains the configuration secrets:"
read -r HEROKU_APP
echo "Enter the Heroku API key to configure the app:"
read -s HEROKU_API_KEY

tokenCheckError=$(curl -s -n https://api.heroku.com/apps/${HEROKU_APP}/config-vars \
  -H "Accept: application/vnd.heroku+json; version=3" \
  -H "Authorization: Bearer ${READ_PROTECTED_TOKEN}" | jq -r '.id')
if [[ ${tokenCheckError} != "null" ]]; then
  echo "Unable to authenticate with Heroku, received error '${tokenCheckError}'. Exiting now"
  exit 1
fi

# use ~ as a delimiter
sed "s~{{.HEROKU_APP}}~${HEROKU_APP}~" ${archive_path}/install/pi-alarm.service.tmpl \
  | sed "s~{{.HEROKU_API_KEY}}~${HEROKU_API_KEY}~" \
  > ${archive_path}/install/pi-alarm.service

sudo mv ${archive_path}/install/pi-alarm.service /etc/systemd/system/
rm ${archive_path}/install/pi-alarm.service.tmpl

install_pagekite

sudo systemctl enable pi-alarm.service
sudo systemctl start pi-alarm.service
sudo systemctl enable pagekite.service
sudo systemctl start pagekite.service

rm -rf ${archive_path}
echo "Installation complete"
