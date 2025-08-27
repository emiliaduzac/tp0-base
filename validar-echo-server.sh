#!/bin/bash

# variables
SERVER="server"
PORT=12345
MESSAGE="Hello, World!"
NETWORK="tp0_testing_net"

# create a temporal container to test the echo server using netcat
RESULT="$(docker run --rm --network="$NETWORK" busybox sh -c "echo \"$MESSAGE\" | nc $SERVER $PORT" || true)"

# validate result
if [ "$RESULT" = "$MESSAGE" ]; then
  echo "action: test_echo_server | result: success"
else
  echo "action: test_echo_server | result: fail"
fi