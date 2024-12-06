SHELL := /bin/bash

IS_WSL := $(shell grep -qi microsoft /proc/version && echo yes || echo no)

# Default target
all: install build

install:
	@echo "Checking and installing dependencies for simple setup..."
	@sudo ./setup.sh --simple-setup --check && \
		echo "All dependencies are installed." || \
		sudo ./setup.sh --simple-setup

install-dev:
ifeq ($(IS_WSL), yes)
	@echo "Detected WSL environment. Checking WSL dependencies..."
	@sudo ./setup.sh --wsl --check && \
		echo "All dependencies are installed." || \
		sudo ./setup.sh --wsl
else
	@echo "Detected Ubuntu 22 environment. Checking Ubuntu dependencies..."
	@sudo ./setup.sh --ubuntu22 --check && \
		echo "All dependencies are installed." || \
		sudo ./setup.sh --ubuntu22
endif

build:
	@echo "Building all components..."
	@(cd bot && go mod tidy)
	@(cd changeset-grpc && npm install)
	@if [ ! -d "transcription-service/.venv" ]; then \
		python3 -m venv transcription-service/.venv; \
	fi
	@(cd transcription-service && source .venv/bin/activate && pip install -r requirements.txt && deactivate)

run: install stop
	@echo "Running Docker Compose in detached mode..."
	@docker compose up -d

run-dev: install-dev stop-dev build
	@echo "Starting services in screen sessions..."
	@mkdir -p logs
	@screen -dmS bot bash -c "cd bot && export $$(cat ../.env | xargs) && go run . 2>&1 | tee ../logs/bot.log"
	@screen -dmS changeset-grpc bash -c "cd changeset-grpc && export $$(cat ../.env | xargs) && npm run start 2>&1 | tee ../logs/changeset-grpc.log"
	@screen -dmS transcription-service bash -c "cd transcription-service && source .venv/bin/activate && export $$(cat ../.env | xargs) && python app.py 2>&1 | tee ../logs/transcription-service.log"
	# Assuming you intended to start only the translation-service and prometheus services from docker-compose-dev.yml
	@screen -dmS translation-service bash -c "docker compose -f docker-compose-dev.yml up translation-service 2>&1 | tee logs/translation-service.log"
	@screen -dmS prometheus bash -c "docker compose -f docker-compose-dev.yml up prometheus 2>&1 | tee logs/prometheus.log"

run-dev-docker: install-dev stop-dev-docker
	@echo "Running development docker-compose environment in detached mode..."
	@docker compose -f docker-compose-dev.yml up -d

stop:
	@echo "Stopping all Docker Compose containers..."
	@docker compose down

stop-dev:
	@echo "Stopping screen sessions..."
	@screen -ls | grep "\.bot" | awk '{print $$1}' | xargs -r -I{} screen -S {} -X quit
	@screen -ls | grep "\.changeset-grpc" | awk '{print $$1}' | xargs -r -I{} screen -S {} -X quit
	@screen -ls | grep "\.transcription-service" | awk '{print $$1}' | xargs -r -I{} screen -S {} -X quit
	@echo "Stopping translation-service and prometheus..."
	@docker compose -f docker-compose-dev.yml down

stop-dev-docker:
	@echo "Stopping development Docker environment..."
	@docker compose -f docker-compose-dev.yml down
