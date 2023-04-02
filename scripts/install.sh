#!/bin/bash

# Install task.
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d

# Install MongoDB tools.
task mongo-tools-setup