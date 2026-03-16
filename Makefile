SHELL := /bin/bash

.PHONY: dev

dev:
	@set -euo pipefail; \
	backend_port="$${PORT:-}"; \
	if [[ -z "$$backend_port" ]]; then backend_port="$$(grep -E '^PORT=' backend/.env | head -n 1 | cut -d '=' -f 2- || true)"; fi; \
	if [[ -z "$$backend_port" ]]; then backend_port="8080"; fi; \
	api_base="$${NEXT_PUBLIC_API_BASE_URL:-http://127.0.0.1:$$backend_port}"; \
	echo "[dev] backend port: $$backend_port"; \
	echo "[dev] NEXT_PUBLIC_API_BASE_URL: $$api_base"; \
	kill_tree() { \
		local pid="$${1:-}"; \
		if [[ -z "$$pid" ]]; then return 0; fi; \
		if ! kill -0 "$$pid" >/dev/null 2>&1; then return 0; fi; \
		local child; \
		for child in $$(pgrep -P "$$pid" 2>/dev/null || true); do \
			kill_tree "$$child"; \
		done; \
		kill "$$pid" >/dev/null 2>&1 || true; \
	}; \
	cleanup() { \
		if [[ -n "$${backend_pid:-}" ]]; then kill_tree "$$backend_pid"; fi; \
		if [[ -n "$${web_pid:-}" ]]; then kill_tree "$$web_pid"; fi; \
		wait "$${backend_pid:-}" >/dev/null 2>&1 || true; \
		wait "$${web_pid:-}" >/dev/null 2>&1 || true; \
	}; \
	trap cleanup EXIT INT TERM; \
	next_lock="web/.next/dev/lock"; \
	if [[ -f "$$next_lock" ]]; then \
		lock_pids="$$(lsof -t "$$next_lock" 2>/dev/null | sort -u || true)"; \
		if [[ -n "$$lock_pids" ]]; then \
			echo "[dev] stopping existing Next.js process using $$next_lock: $$lock_pids"; \
			for pid in $$lock_pids; do \
				kill_tree "$$pid"; \
			done; \
			sleep 1; \
		else \
			echo "[dev] removing stale Next.js lock: $$next_lock"; \
		fi; \
		rm -f "$$next_lock"; \
	fi; \
	( cd backend && PORT="$$backend_port" make run ) & backend_pid="$$!"; \
	( cd web && env -u PORT NEXT_PUBLIC_API_BASE_URL="$$api_base" pnpm dev ) & web_pid="$$!"; \
	status=0; \
	while true; do \
		if ! kill -0 "$$backend_pid" >/dev/null 2>&1; then \
			wait "$$backend_pid" || status="$$?"; \
			echo "[dev] backend stopped"; \
			break; \
		fi; \
		if ! kill -0 "$$web_pid" >/dev/null 2>&1; then \
			wait "$$web_pid" || status="$$?"; \
			echo "[dev] web stopped"; \
			break; \
		fi; \
		sleep 1; \
	done; \
	exit "$$status"
