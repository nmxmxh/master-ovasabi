#!/bin/bash

# OVASABI Yarn Setup Script
# This script sets up Yarn for OVASABI documentation tooling

echo "Setting up Yarn for OVASABI documentation..."

# Check if yarn is installed
if ! command -v yarn &> /dev/null; then
    echo "Yarn not found. Installing Yarn..."
    npm install -g yarn
fi

# Create necessary directories
mkdir -p .yarn

# Install dependencies
yarn install

# Add markdown-link-check if needed
if ! yarn list | grep -q markdown-link-check; then
    echo "Adding markdown-link-check..."
    yarn add --dev markdown-link-check
fi

echo "Yarn setup complete. You can now use 'make js-setup', 'make lint', and 'make lint-fix' commands." 