# Usage: make docker-slim-all
# This will run docker-slim on all service images after build

docker-slim-all:
	docker-slim build --http-probe=false master-ovasabi-app || true
	docker-slim build --http-probe=false master-ovasabi-nexus || true
	docker-slim build --http-probe=false master-ovasabi-media-streaming || true
	docker-slim build --http-probe=false master-ovasabi-ws-gateway || true
	docker-slim build --http-probe=false master-ovasabi-nginx || true
	# docker-slim build --http-probe=false master-ovasabi-ai || true

# Comprehensive Docker cleanup - removes orphaned volumes, build cache, and unused resources
docker-cleanup-all:
	@echo "🧹 Starting comprehensive Docker cleanup..."
	@echo "📊 Before cleanup:"
	@docker system df
	@echo ""
	@echo "🗑️  Removing orphaned volumes..."
	docker volume prune -f
	@echo "🗑️  Removing build cache (this will free the most space)..."
	docker builder prune -a -f
	@echo "🗑️  Removing unused images..."
	docker image prune -a -f
	@echo "🗑️  Removing unused containers..."
	docker container prune -f
	@echo "🗑️  Removing unused networks..."
	docker network prune -f
	@echo "🗑️  Removing docker-slim artifacts..."
	docker images | grep '\.slim$$' | awk '{print $$1 ":" $$2}' | xargs -r docker rmi || true
	@echo ""
	@echo "📊 After cleanup:"
	@docker system df
	@echo "✅ Docker cleanup complete!"

# You can add more images as needed. Adjust image names to match your tags.
