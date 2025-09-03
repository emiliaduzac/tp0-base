#!/bin/bash

# base variables in case no args are given
OUTFILE="docker-compose-dev.yaml"
NCLIENTS=5

# if 2 args are given, override the defaults
if [ $# -eq 2 ]; then
  OUTFILE="$1"
  NCLIENTS="$2"
else
  echo "Using defaults: "$OUTFILE" with "$NCLIENTS" clients"
fi

: > "$OUTFILE"

# base of the compose
cat <<EOF >> "$OUTFILE"
name: tp0
services:
  server:
    container_name: server
    image: server:latest
    entrypoint: python3 /main.py
    environment:
      - PYTHONUNBUFFERED=1
      - CLIENT_AMOUNT=${NCLIENTS} 
    networks:
      - testing_net
    volumes:
      - ./server/config.ini:/config.ini
EOF

# add the clients
for i in $(seq 1 "$NCLIENTS"); do
cat <<EOF >> "$OUTFILE" 
  client$i:
    container_name: client$i
    image: client:latest
    entrypoint: /client
    env_file: ./env_variables/client${i}.env
    networks:
      - testing_net
    volumes:
      - ./client/config.yaml:/config.yaml
      - ./.data/agency-${i}.csv:/data/agency-${i}.csv
    depends_on:
      - server

EOF
done

# add the network config
cat <<EOF >> "$OUTFILE"
networks:
  testing_net:
    ipam:
      driver: default
      config:
        - subnet: 172.25.125.0/24

EOF

echo "Generated "$OUTFILE" with "$NCLIENTS" clients"