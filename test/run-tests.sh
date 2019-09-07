#!/bin/sh
set -e

docker-compose -f $PWD/test/integration-test-setup.yaml up --build -d

until curl http://admin:admin@localhost:8444/v2/catalog -H "X-Broker-API-Version: 2.14" --silent --output /dev/null ; do
  >&2 echo "Broker is unavailable - sleeping"
  sleep 3
done

docker run -v $PWD/test/collections:/etc/newman --network="host" -t postman/newman:ubuntu \
    run "integration-tests.postman_collection.json" \
    --environment="OSB-Integration-test.postman_environment.json" \
    --reporters="json,cli" --reporter-json-export="newman-results.json"

docker-compose -f $PWD/test/integration-test-setup.yaml down