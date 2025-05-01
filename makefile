.PHONY: install install-dev build run run-dev run-dev-docker stop stop-dev stop-dev-docker

help: check-not-root
	@echo "Usage: make [option]"
	@echo ""
	@echo "Options:"
	@echo "  run              Run all services"
	@echo "  run-dev          Run all services for development"
	@echo "  run-dev-docker   Run all services for development using docker-compose"
	@echo "  stop             Stop all services"

check-not-root:
	@if [ "$$(id -u)" = "0" ]; then \
	  echo "Error: Do not run this Makefile as root (UID 0)."; \
	  exit 1; \
	fi
	@if [ -n "$${SUDO_USER}" ]; then \
	  echo "Error: Do not run this Makefile via sudo."; \
	  exit 1; \
	fi

check-docker-rights:
	@if [ $$(id -nG | grep -cw docker) -eq 0 ]; then \
		tput clear > /dev/tty; \
		tput cup 0 0 > /dev/tty; \
		tput ed > /dev/tty; \
		echo "\e[1;31m========================= ATTENTION REQUIRED =========================\e[0m" > /dev/tty; \
		echo "\e[1;33mThis user is not in the docker group.\e[0m" > /dev/tty; \
		echo "\e[1;33mPlease follow the steps below:\e[0m" > /dev/tty; \
		echo "\e[1;32m1. Run: \e[1;34msudo usermod -aG docker $$USER\e[0m" > /dev/tty; \
		echo "\e[1;32m2. Reopen your current session.\e[0m" > /dev/tty; \
		echo "\e[1;34m   (If you are using VS Code, restart VS Code)\e[0m" > /dev/tty; \
		echo "\e[1;32m3. Rerun the makefile.\e[0m" > /dev/tty; \
		echo "\e[1;31m======================================================================\e[0m" > /dev/tty; \
		exit 1; \
	fi

install: check-not-root
	@./setup.sh --simple-setup --check || sudo ./setup.sh --simple-setup
	@echo "All dependencies are installed"

install-dev: check-not-root
	@./setup.sh --check || sudo ./setup.sh
	@echo "All dependencies are installed"

build: check-not-root
	@if [ ! "$$(ls -A transcription-service)" ]; then \
		git submodule update --init --recursive; \
		cd changeset-grpc/etherpad-lite; \
		src/bin/installDeps.sh; \
		cd ..; \
		NPM_BIN_PATH=$$(ls -d ~/.nvm/versions/node/*/bin | head -n 1)/npm; \
		if [ -f "$$NPM_BIN_PATH" ]; then \
			$$NPM_BIN_PATH install; \
		else \
			echo "npm not found in $$NPM_BIN_PATH"; \
			exit 1; \
		fi; \
	fi
	@if [ ! "$$(ls -A changeset-grpc)" ]; then \
		git submodule update --init --recursive; \
		cd changeset-grpc/etherpad-lite; \
		src/bin/installDeps.sh; \
		cd ..; \
		NPM_BIN_PATH=$$(ls -d ~/.nvm/versions/node/*/bin | head -n 1)/npm; \
		if [ -f "$$NPM_BIN_PATH" ]; then \
			$$NPM_BIN_PATH install; \
		else \
			echo "npm not found in $$NPM_BIN_PATH"; \
			exit 1; \
		fi; \
	fi

	@cd bot && go mod tidy
	
	@cd transcription-service && if [ ! -d ".venv" ]; then \
		python3 -m venv .venv && \
		bash -c "source .venv/bin/activate && pip install -r requirements.txt && deactivate"; \
	fi

generate-env-files: check-not-root
	@if [ ! -f .env -o ! -f .env-dev -o ! -f .env-dev-docker ]; then \
		./generate-env.sh; \
	fi


run: check-not-root generate-env-files install check-docker-rights stop
	@docker compose up -d

	@screen -dmS bot bash -c "docker logs -f bot 2>&1 | tee logs/bot.log"
	@screen -dmS changeset-service bash -c "docker logs -f changeset-service 2>&1 | tee logs/changeset-service.log"
	@screen -dmS transcription-service bash -c "docker logs -f transcription-service 2>&1 | tee logs/transcription-service.log"
	@screen -dmS translation-service bash -c "docker logs -f translation-service 2>&1 | tee logs/translation-service.log"
	@screen -dmS prometheus bash -c "docker logs -f prometheus 2>&1 | tee logs/prometheus.log"

	@echo "------------------------------------------------------"
	@echo "All services are running in the background."
	@echo "The logs are available in the logs/ directory."
	@echo "------------------------------------------------------"

run-dev: check-not-root generate-env-files install-dev check-docker-rights stop build
	@docker compose -f docker-compose-dev.yml up --no-start
	@screen -dmS bot bash -c "cd bot && set -a && source ../.env-dev && set +a && go run . 2>&1 | tee ../logs/bot.log"
	
	@screen -dmS changeset-service bash -c '\
		NPM_BIN_PATH=$$(ls -d $$HOME/.nvm/versions/node/*/bin | head -n 1)/npm; \
		if [ -f "$$NPM_BIN_PATH" ]; then \
			echo "npm found at $$NPM_BIN_PATH"; \
			cd changeset-grpc && \
			set -a && source ../.env-dev && set +a && \
			"$$NPM_BIN_PATH" run start 2>&1 | tee ../logs/changeset-service.log; \
		else \
			echo "npm not found at $$NPM_BIN_PATH"; \
			exit 1; \
		fi'

	
	@screen -dmS transcription-service bash -c "cd transcription-service && source .venv/bin/activate && set -a && source ../.env-dev && set +a && python main.py 2>&1 | tee ../logs/transcription-service.log"
	@screen -dmS translation-service bash -c "docker compose -f docker-compose-dev.yml up translation-service 2>&1 | tee logs/translation-service.log"
	@screen -dmS prometheus bash -c "docker compose -f docker-compose-dev.yml up prometheus  2>&1 | tee logs/prometheus.log"

	@echo "------------------------------------------------------"
	@echo "All services are running in the background."
	@echo "The logs are available in the logs/ directory."
	@echo "More detailed logs of the transcription-service are available in transcription-service/logs/."
	@echo "------------------------------------------------------"

run-dev-docker: check-not-root generate-env-files install-dev check-docker-rights stop build
	@docker compose -f docker-compose-dev.yml up -d --build

	@screen -dmS bot bash -c "docker logs -f bot 2>&1 | tee logs/bot.log"
	@screen -dmS changeset-service bash -c "docker logs -f changeset-service 2>&1 | tee logs/changeset-service.log"
	@screen -dmS transcription-service bash -c "docker logs -f transcription-service 2>&1 | tee logs/transcription-service.log"
	@screen -dmS translation-service bash -c "docker logs -f translation-service 2>&1 | tee logs/translation-service.log"
	@screen -dmS prometheus bash -c "docker logs -f prometheus 2>&1 | tee logs/prometheus.log"

	@echo "------------------------------------------------------"
	@echo "All services are running in the background."
	@echo "The logs are available in the logs/ directory."
	@echo "More detailed logs of the transcription-service are available in transcription-service/logs/."
	@echo "------------------------------------------------------"

stop: check-not-root check-docker-rights stop-dev stop-dev-docker
	@docker compose down

stop-dev: check-not-root check-docker-rights
	@for service in bot changeset-service transcription-service; do \
		screen -ls | grep ".$$service" | awk '{print $$1}' | while read session; do \
			echo "Stopping screen session $$session..."; \
			screen -S $$session -X quit; \
		done; \
	done
	@echo "Stopping translation-service..."; docker compose -f docker-compose-dev.yml down translation-service
	@echo "Stopping prometheus..."; docker compose -f docker-compose-dev.yml down prometheus

stop-dev-docker: check-not-root check-docker-rights
	@docker compose -f docker-compose-dev.yml down
