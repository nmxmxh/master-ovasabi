#!/bin/bash

# Build dynamic service registration tools
# This script builds the dynamic service registration generator and inspector tools

set -e

echo "Building dynamic service registration tools..."

# Create bin directory if it doesn't exist
mkdir -p bin

# Build the generator tool
echo "Building service registration generator..."
go build -o bin/service-registration-generator ./pkg/registration/cmd/generate

# Build the dynamic registry inspector
echo "Building dynamic registry inspector..."
go build -o bin/registry-inspect-dynamic ./pkg/registration/cmd/registry-inspect-dynamic

# Make them executable
chmod +x bin/service-registration-generator
chmod +x bin/registry-inspect-dynamic

echo "Tools built successfully!"
echo ""
echo "Available tools:"
echo "  bin/service-registration-generator - Generate service registration configs from proto files"
echo "  bin/registry-inspect-dynamic      - Enhanced registry inspection with dynamic capabilities"
echo ""
echo "Usage examples:"
echo "  # Generate service registration from proto files"
echo "  ./bin/service-registration-generator -proto-path api/protos -output config/service_registration_generated.json"
echo ""
echo "  # Inspect all services"
echo "  ./bin/registry-inspect-dynamic -mode services"
echo ""
echo "  # Generate dynamic service registration"
echo "  ./bin/registry-inspect-dynamic -mode generate -output config/service_registration_dynamic.json"
echo ""
echo "  # Inspect specific service"
echo "  ./bin/registry-inspect-dynamic -mode inspect -service user"
echo ""
echo "  # Validate service configuration"
echo "  ./bin/registry-inspect-dynamic -mode validate -service user"
echo ""
echo "  # Compare two services"
echo "  ./bin/registry-inspect-dynamic -mode compare -service user -compare admin"
echo ""
echo "  # Export service dependency graph"
echo "  ./bin/registry-inspect-dynamic -mode graph -output service_graph.json"
