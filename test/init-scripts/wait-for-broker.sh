#!/bin/sh
set -e

until nc -z broker 884; do
  >&2 echo "Broker is unavailable - sleeping"
  sleep 1
done

exec ./servicebroker-linux