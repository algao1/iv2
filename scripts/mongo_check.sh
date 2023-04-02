#!/bin/bash

export $(grep -v '^#' .env | xargs -d '\n')

while IFS= read -r line; do
  match=$(echo $line | grep -oP '(?<=(username|password): ).*')
  if [[ ! -z "$match" && "$match" == "$MONGO_USERNAME" ]]; then
    username_set=true
  elif [[ ! -z "$match" && "$match" == "$MONGO_PASSWORD" ]]; then
    password_set=true
  fi
done < "$1"

if [[ ! $username_set ]]; then
  echo "[FAIL] mongo username mismatch in .env and config"
  exit 1
elif [[ ! $password_set ]]; then
  echo "[FAIL] mongo password mismatch in .env and config"
  exit 1
else
  echo "[SUCCESS] mongo username and password matches in .env and config"
fi