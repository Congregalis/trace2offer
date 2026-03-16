SHELL := /bin/bash

GO ?= go
PORT ?= 8080
DATA_DIR ?= ./data
API_BASE ?= http://127.0.0.1:$(PORT)

.PHONY: help run test build smoke

help:
	@echo "Targets:"
	@echo "  make run    - 启动后端服务 (默认读取 .env，环境变量优先)"
	@echo "  make test   - 运行 Go 测试"
	@echo "  make build  - 编译后端二进制到 ./server"
	@echo "  make smoke  - 冒烟测试: 自动启动服务并验证全部 API"

run:
	$(GO) run ./cmd/server

test:
	$(GO) test ./...

build:
	$(GO) build -o server ./cmd/server

smoke:
	@set -euo pipefail; \
	tmp_data_dir="$$(mktemp -d)"; \
	tmp_bin_path="$$(mktemp /tmp/trace2offer_smoke_server.XXXXXX)"; \
	port="$${SMOKE_PORT:-$$(awk 'BEGIN{srand(); print int(30000 + rand() * 2000)}')}"; \
	server_pid=""; \
	cleanup() { \
		if [[ -n "$$server_pid" ]]; then \
			kill "$$server_pid" >/dev/null 2>&1 || true; \
			wait "$$server_pid" >/dev/null 2>&1 || true; \
		fi; \
		rm -rf "$$tmp_data_dir"; \
		rm -f "$$tmp_bin_path"; \
	}; \
	trap cleanup EXIT; \
	echo "[smoke] starting server on $$port with temp data dir: $$tmp_data_dir"; \
	$(GO) build -o "$$tmp_bin_path" ./cmd/server; \
	PORT="$$port" T2O_DATA_DIR="$$tmp_data_dir" OPENAI_API_KEY="smoke_dummy_key" "$$tmp_bin_path" >/tmp/trace2offer_smoke_server.log 2>&1 & \
	server_pid="$$!"; \
	for _ in $$(seq 1 80); do \
		if curl -fsS "http://127.0.0.1:$$port/health" >/dev/null 2>&1; then \
			break; \
		fi; \
		sleep 0.2; \
	done; \
	if ! curl -fsS "http://127.0.0.1:$$port/health" >/dev/null 2>&1; then \
		echo "[smoke] server not ready"; \
		echo "----- server log -----"; \
		cat /tmp/trace2offer_smoke_server.log; \
		exit 1; \
	fi; \
	response_file="/tmp/trace2offer_smoke_response.json"; \
	status_code="$$(curl -sS -o "$$response_file" -w '%{http_code}' "http://127.0.0.1:$$port/api/leads")"; \
	test "$$status_code" = "200"; \
	echo "[smoke] GET /api/leads => $$status_code"; \
	status_code="$$(curl -sS -o "$$response_file" -w '%{http_code}' -X POST "http://127.0.0.1:$$port/api/leads" \
		-H 'Content-Type: application/json' \
		-d '{"company":"Smoke Inc","position":"Backend Engineer","source":"Smoke","status":"new","priority":3,"next_action":"follow up","notes":"smoke","company_website_url":"https://example.com","jd_url":"https://example.com/jobs/1","location":"Taipei"}')"; \
	test "$$status_code" = "201"; \
	echo "[smoke] POST /api/leads => $$status_code"; \
	lead_id="$$(sed -n 's/.*"id":"\([^"]*\)".*/\1/p' "$$response_file" | head -n 1)"; \
	if [[ -z "$$lead_id" ]]; then \
		echo "[smoke] failed to parse created lead id"; \
		cat "$$response_file"; \
		exit 1; \
	fi; \
	status_code="$$(curl -sS -o "$$response_file" -w '%{http_code}' -X PUT "http://127.0.0.1:$$port/api/leads/$$lead_id" \
		-H 'Content-Type: application/json' \
		-d '{"company":"Smoke Inc","position":"Backend Engineer II","source":"Smoke","status":"preparing","priority":4,"next_action":"phone screen","notes":"put","company_website_url":"https://example.com","jd_url":"https://example.com/jobs/2","location":"Taipei"}')"; \
	test "$$status_code" = "200"; \
	echo "[smoke] PUT /api/leads/:id => $$status_code"; \
	status_code="$$(curl -sS -o "$$response_file" -w '%{http_code}' -X PATCH "http://127.0.0.1:$$port/api/leads/$$lead_id" \
		-H 'Content-Type: application/json' \
		-d '{"company":"Smoke Inc","position":"Backend Engineer II","source":"Smoke","status":"interviewing","priority":5,"next_action":"onsite","notes":"patch","company_website_url":"https://example.com","jd_url":"https://example.com/jobs/3","location":"Remote"}')"; \
	test "$$status_code" = "200"; \
	echo "[smoke] PATCH /api/leads/:id => $$status_code"; \
	status_code="$$(curl -sS -o "$$response_file" -w '%{http_code}' "http://127.0.0.1:$$port/api/agent/settings")"; \
	test "$$status_code" = "200"; \
	echo "[smoke] GET /api/agent/settings => $$status_code"; \
	status_code="$$(curl -sS -o "$$response_file" -w '%{http_code}' -X PATCH "http://127.0.0.1:$$port/api/agent/settings" \
		-H 'Content-Type: application/json' \
		-d '{"max_steps":5}')"; \
	test "$$status_code" = "200"; \
	echo "[smoke] PATCH /api/agent/settings => $$status_code"; \
	status_code="$$(curl -sS -o "$$response_file" -w '%{http_code}' -X DELETE "http://127.0.0.1:$$port/api/leads/$$lead_id")"; \
	test "$$status_code" = "204"; \
	echo "[smoke] DELETE /api/leads/:id => $$status_code"; \
	echo "[smoke] all checks passed"
