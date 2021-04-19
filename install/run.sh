#!/bin/bash


if [[ -z ${HEROKU_APP} ]]; then
  echo "HEROKU_APP env var not set, exiting now"
  exit 1
fi

if [[ -z ${HEROKU_API_KEY} ]]; then
  echo "HEROKU_API_KEY env var not set, exiting now"
  exit 1
fi

vars=$(curl -s -n https://api.heroku.com/apps/${HEROKU_APP}/config-vars \
  -H "Accept: application/vnd.heroku+json; version=3" \
  -H "Authorization: Bearer ${HEROKU_API_KEY}" \
  | jq -r 'to_entries[] | "\(.key)=\(.value)"')

for var in ${vars}; do
  export "${var}"
done

/home/pi/pi-alarm
