#!/bin/bash

# Project Structure Validation Script
# Validates that all moved files are in correct locations and references are updated

echo "🔍 Validating Project Structure Cleanup..."
echo

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

errors=0
warnings=0

# Function to check if file exists
check_file() {
    if [ -f "$1" ]; then
        echo -e "${GREEN}✅ Found: $1${NC}"
    else
        echo -e "${RED}❌ Missing: $1${NC}"
        ((errors++))
    fi
}

# Function to check if file is NOT in old location
check_not_exists() {
    if [ ! -f "$1" ]; then
        echo -e "${GREEN}✅ Correctly removed: $1${NC}"
    else
        echo -e "${YELLOW}⚠️  Still exists (should be removed): $1${NC}"
        ((warnings++))
    fi
}

# Function to check if string exists in file
check_string_in_file() {
    if grep -q "$2" "$1" 2>/dev/null; then
        echo -e "${GREEN}✅ Found '$2' in $1${NC}"
    else
        echo -e "${RED}❌ Missing '$2' in $1${NC}"
        ((errors++))
    fi
}

echo "📁 Checking moved files are in new locations..."
check_file "deployments/docker/redis.conf"
check_file "config/service_registration.json"
check_file "deployments/sample.env"
check_file "docs/mkdocs.yml"
check_file "docs/development/gemini-guide.md"
check_file "deployments/docker/slim.report.json"
check_file "scripts/setup-yarn.sh"

echo
echo "🗑️  Checking old files are removed..."
check_not_exists "redis.conf"
check_not_exists "service_registration.json"
check_not_exists "sample.env"
check_not_exists "mkdocs.yml"
check_not_exists "gemini.md"
check_not_exists "slim.report.json"
check_not_exists "setup-yarn.sh"

echo
echo "🔗 Checking updated references..."
check_string_in_file "deployments/docker/docker-compose.yml" "./redis.conf"
check_string_in_file "internal/bootstrap/services.go" "config/service_registration.json"
check_string_in_file "deployments/docker/Dockerfile" "/config/service_registration.json"
check_string_in_file "scripts/generate_service_registration.sh" "config/service_registration.json"

echo
echo "🏗️  Checking PostgreSQL 18 files are intact..."
check_file "deployments/docker/postgresql18.conf"
check_file "deployments/docker/Dockerfile.postgres18"
check_file "deployments/docker/02-optimize-pg18.sql"
check_file "internal/repository/enhanced_pg18.go"
check_file "database/migrations/000035_postgresql_18_virtual_columns.sql"
check_file "database/migrations/000036_postgresql_18_full_optimization.sql"

echo
echo "📊 Validation Summary:"
if [ $errors -eq 0 ] && [ $warnings -eq 0 ]; then
    echo -e "${GREEN}🎉 All checks passed! Project structure cleanup is successful.${NC}"
    exit 0
elif [ $errors -eq 0 ]; then
    echo -e "${YELLOW}⚠️  $warnings warnings found, but no critical errors.${NC}"
    exit 0
else
    echo -e "${RED}❌ $errors errors found. Please fix the issues above.${NC}"
    exit 1
fi
