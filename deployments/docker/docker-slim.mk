# Usage: make docker-slim-all
# This will run docker-slim on all service images after build

docker-slim-all:
	# docker-slim build --http-probe=false ovasabi/master-ovasabi-ai || true
	docker-slim build --http-probe=false ovasabi/master-ovasabi-nexus || true
	docker-slim build --http-probe=false ovasabi/master-ovasabi-media-streaming || true
	docker-slim build --http-probe=false ovasabi/master-ovasabi-ws-gateway || true
	docker-slim build --http-probe=false ovasabi/master-ovasabi-wasm || true

# You can add more images as needed. Adjust image names to match your tags.
