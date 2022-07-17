#!/bin/bash

docker-compose down
docker-compose pull
docker-compose up --env-file iv2.env -d
docker image prune