.PHONY: install install-dev build run run-dev run-dev-docker stop stop-dev stop-dev-docker

# Define flags
IS_WSL := $(shell grep -qi microsoft /proc/version && echo yes || echo no)

all: install

install:
ifeq ($(IS_WSL), yes)
	@sudo ./setup.sh --wsl --check || sudo ./setup.sh --wsl
else
	@sudo ./setup.sh --simple-setup --check || sudo ./setup.sh --simple-setup
endif
	@echo "All dependencies are installed"

install-dev:
ifeq ($(IS_WSL), yes)
	@sudo ./setup.sh --wsl --check || sudo ./setup.sh --wsl
else
	@sudo ./setup.sh --ubuntu22 --check || sudo ./setup.sh --ubuntu22
endif
	@echo "All dependencies are installed"

build:
	@cd bot && go mod tidy
	@cd changeset-grpc && npm install && cd etherpad-lite && src/bin/installDeps.sh
	@cd transcription-service && \
		[ -d ".venv" ] || python3 -m venv .venv && \
		bash -c "source .venv/bin/activate && pip install -r requirements.txt && deactivate"


run: install stop
	@docker compose up -d

run-dev: install-dev stop-dev build
	@screen -dmS bot bash -c "cd bot && export $(cat ../.env | xargs) && go run . 2>&1 | tee ../logs/bot.log"
	@screen -dmS changeset-grpc bash -c "cd changeset-grpc && export $(cat ../.env | xargs) && npm run start 2>&1 | tee ../logs/changeset-grpc.log"
	@screen -dmS transcription-service bash -c "cd transcription-service && source .venv/bin/activate && export $(cat ../.env | xargs) && python main.py 2>&1 | tee ../logs/transcription-service.log"
	@screen -dmS translation-service bash -c "docker compose -f docker-compose-dev.yml up translation-service 2>&1 | tee logs/translation-service.log"
	@screen -dmS prometheus bash -c "docker compose -f docker-compose-dev.yml up prometheus  2>&1 | tee logs/prometheus.log"

run-dev-docker: install-dev stop-dev-docker
	@docker compose -f docker-compose-dev.yml up -d

stop: stop-dev stop-dev-docker
	@docker compose down

stop-dev:
	@for service in bot changeset-grpc transcription-service; do \
		screen -ls | grep ".$$service" | awk '{print $$1}' | while read session; do \
			echo "Stopping screen session $$session..."; \
			screen -S $$session -X quit; \
		done; \
	done
	@echo "Stopping translation-service..."; docker compose -f docker-compose-dev.yml down translation-service
	@echo "Stopping prometheus..."; docker compose -f docker-compose-dev.yml down prometheus

stop-dev-docker:
	@docker compose -f docker-compose-dev.yml down
