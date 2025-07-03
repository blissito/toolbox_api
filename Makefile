.PHONY: build run stop clean rebuild

build:
	podman build -t toolbox-api .

run: stop build
	podman run -d --env-file=.env -p 8000:8000 --name toolbox-api toolbox-api
	@echo "\nAPI running at http://localhost:8000"

# Find and kill process using port 8000
kill-port:
	@echo "Checking for processes using port 8000..."
	@lsof -ti :8000 | xargs -r kill -9 2>/dev/null || true

stop: kill-port
	podman machine stop
	podman machine start
	@echo "Stopping and removing existing containers..."
	@podman stop toolbox-api 2>/dev/null || true
	@podman rm -f toolbox-api 2>/dev/null || true

clean: stop
	@echo "Removing container images..."
	@podman rmi toolbox-api 2>/dev/null || true

# Rebuild: stop existing containers, clean up, and run a fresh build
rebuild: stop clean run
