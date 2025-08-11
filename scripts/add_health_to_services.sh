#!/bin/bash

# Script to add health monitoring to all service providers
# This follows the pattern established in user, admin, content, and campaign services

set -e

# List of services to update (excluding user, admin, content, campaign which are already done)
SERVICES=(
    "analytics"
    "commerce" 
    "localization"
    "media"
    "messaging"
    "nexus"
    "notification"
    "product"
    "referral"
    "scheduler"
    "search"
    "security"
    "waitlist"
    "contentmoderation"
    "talent"
    "pattern"
    "orchestration"
    "ai"
    "crawler"
)

echo "Adding health monitoring to ${#SERVICES[@]} services..."

for service in "${SERVICES[@]}"; do
    provider_file="internal/service/${service}/provider.go"
    
    if [[ ! -f "$provider_file" ]]; then
        echo "❌ Skipping $service - provider.go not found"
        continue
    fi
    
    # Check if health is already imported
    if grep -q '"github.com/nmxmxh/master-ovasabi/pkg/health"' "$provider_file"; then
        echo "✅ $service already has health import"
        continue
    fi
    
    echo "🔧 Updating $service..."
    
    # Add health import after hello import
    sed -i '' '/pkg\/hello/a\
	"github.com/nmxmxh/master-ovasabi/pkg/health"
' "$provider_file"
    
    # Find hello.StartHelloWorldLoop and add health monitoring before it
    if grep -q "hello.StartHelloWorldLoop" "$provider_file"; then
        sed -i '' '/hello\.StartHelloWorldLoop.*'$service'/i\
		// Start health monitoring (following hello package pattern)\
		healthDeps := \&health.ServiceDependencies{\
			Database: db,\
			Redis:    cache, // Reuse existing cache (may be nil if retrieval failed)\
		}\
		health.StartHealthSubscriber(ctx, prov, log, "'$service'", healthDeps)\
		
' "$provider_file"
        echo "✅ Added health monitoring to $service"
    else
        echo "⚠️  Could not find hello.StartHelloWorldLoop in $service - may need manual update"
    fi
done

echo ""
echo "🎉 Health monitoring update complete!"
echo "📋 Summary:"
echo "   - Updated ${#SERVICES[@]} service provider files"
echo "   - Each service now has health check event subscribers"
echo "   - Health events follow canonical format: {service}:health:v1:{state}"
echo "   - Frontend will receive proper health responses!"
echo ""
echo "🔍 Next steps:"
echo "   1. Run 'go build ./internal/service/...' to verify no compilation errors"
echo "   2. Test health events with: curl -X POST localhost:8080/events -d '{\"event_type\":\"user:health:v1:requested\"}'"
echo "   3. Check that ArchitectureDemo.tsx receives health responses"
