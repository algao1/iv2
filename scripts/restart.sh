#!/bin/bash

export $(grep -v '^#' iv2.env | xargs)
mongodump -u=${MONGO_USERNAME} -p=${MONGO_PASSWORD} -o "dump_$(date +'%m_%d_%Y_%H:%M')"

doctl registry login --expiry-seconds 600
docker-compose down
docker-compose pull
docker-compose --env-file iv2.env up -d
echo y | docker image prune