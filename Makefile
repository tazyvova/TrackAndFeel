.PHONY: up down logs dev test lint

up:
	docker compose up -d
	@echo "✅ Stack is up:"
	@echo "   Frontend: http://localhost:5173"
	@echo "   Backend : http://localhost:8080/healthz"
	@echo "   DB      : localhost:5432 (postgres/postgres, db=training)"

logs:
	docker compose logs -f --tail=100

down:
	docker compose down

# convenience: bring up + follow logs (Ctrl+C stops log tail, stack keeps running)
dev: up logs

test:
	docker compose -f docker-compose.yml -f docker-compose.test.yml run --rm backend-test
	# add frontend tests later

lint:
	cd backend && golangci-lint run
	cd frontend && npm run lint || true
