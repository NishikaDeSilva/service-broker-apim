#!/bin/sh
set -e

docker-compose -f $PWD/test/docker-compose.yaml up --build -d

until nc -z localhost 8444; do
  >&2 echo "Broker is unavailable - sleeping"
  sleep 3
done

docker run -v $PWD/test/collections:/etc/newman --network="host" -t postman/newman:ubuntu \
    run "integration-tests.postman_collection.json" \
    --environment="OSB-Integration-test.postman_environment.json" \
    --reporters="json,cli" --reporter-json-export="newman-results.json"

docker-compose -f $PWD/test/docker-compose.yaml down