#!/bin/bash

set -e

dirPath=$(dirname $(readlink -f "${BASH_SOURCE}"))

if ! command -v ocb 2>&1 >/dev/null
then
    echo "ocb could not be found, installing it now"
	curl --proto '=https' --tlsv1.2 -fL -o ocb \
https://github.com/open-telemetry/opentelemetry-collector-releases/releases/download/cmd%2Fbuilder%2Fv0.116.0/ocb_0.116.1_linux_amd64
    chmod +x ocb
	mkdir -p ~/.local/bin
	mv ./ocb ~/.local/bin
fi

echo "Compiling..."
ocb --config $dirPath/builder-config.yaml

echo "Starting collector..."
chmod +x ./otelcol-dev/otelcontribcol
./otelcol-dev/otelcontribcol --config collector-config.yaml --feature-gates=service.profilesSupport
