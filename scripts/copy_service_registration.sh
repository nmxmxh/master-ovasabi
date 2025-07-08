#!/bin/sh
set -e

# Copy the config/service_registration.json into the build context for embedding or runtime use
mkdir -p ./config
cp ../../config/service_registration.json ./config/service_registration.json
