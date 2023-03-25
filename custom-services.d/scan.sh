#!/usr/bin/with-contenv bash

if [ -z "$OCCPATH" ]; then
      echo "Error: OCCPATH is not defined as environment variable."
      sleep 60
      exit 1
fi

if [ -z "$SYNCTHING_URL" ]; then
      echo "Error: SYNCTHING_URL is not defined as environment variable."
      sleep 60
      exit 1
fi

FLAGS_ARRAY=(-occpath="$OCCPATH" -target="$SYNCTHING_URL")

if [ "$SKIP_SYNCTHING_SSL_VALIDATION" != "0" ]; then
    FLAGS_ARRAY+=(-i=1)
fi

# Add the mapping from env variables starting with NC_ST_MAPPING_
for var in "${!NC_ST_MAPPING_@}"; do
    FLAGS_ARRAY+=(-mapping="${!var}")
done

SYNCTHING_API_KEY_FILE=/config/syncthing_api_key.txt

# Read api key from file (which only root should be able to read)
# Keep waiting until this api key file exists, then continue
while [ ! -f "$SYNCTHING_API_KEY_FILE" ]; do
    echo "Please put the Syncthing API key in this text file: $SYNCTHING_API_KEY_FILE"
    sleep 60
done

# shellcheck disable=SC2024
sudo -u abc -s /usr/bin/with-contenv /scanner/scanner -apikey-from-stdin "${FLAGS_ARRAY[@]}" < "$SYNCTHING_API_KEY_FILE" 2>&1