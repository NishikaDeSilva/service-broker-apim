#!/bin/sh
set -e

until nc -z mysql 3306; do
  >&2 echo "Mysql is unavailable - sleeping"
  sleep 1
done

until nc -z wso2apim 9443; do
  >&2 echo "WSO2 APIM is unavailable - sleeping"
  sleep 1
done


exec ./servicebroker-linux