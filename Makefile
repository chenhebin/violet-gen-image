COMPOSE := docker compose --env-file .env.deploy -f compose.production.yaml

.PHONY: help deploy deploy-check deploy-build deploy-migrate deploy-bootstrap-admin deploy-up deploy-ps deploy-logs deploy-restart deploy-down

help:
	@echo "make deploy                 Build, migrate and start the production stack"
	@echo "make deploy-bootstrap-admin Create the first platform admin (one time)"
	@echo "make deploy-ps              Show container status"
	@echo "make deploy-logs            Follow application logs"
	@echo "make deploy-restart         Restart application containers"
	@echo "make deploy-down            Stop containers without deleting data"

deploy: deploy-check deploy-build deploy-migrate deploy-up

deploy-check:
	@test -f .env.deploy || (echo "Missing .env.deploy; copy .env.deploy.example and fill every required value" && exit 1)
	@$(COMPOSE) config --quiet

deploy-build:
	@$(COMPOSE) build api admin client

deploy-migrate:
	@$(COMPOSE) --profile tools run --rm migrate

deploy-bootstrap-admin: deploy-check
	@test -n "$$BOOTSTRAP_ADMIN_EMAIL" || (echo "BOOTSTRAP_ADMIN_EMAIL is required" && exit 1)
	@test -n "$$BOOTSTRAP_ADMIN_PASSWORD" || (echo "BOOTSTRAP_ADMIN_PASSWORD is required" && exit 1)
	@$(COMPOSE) --profile tools run --rm bootstrap-admin

deploy-up:
	@$(COMPOSE) up -d postgres minio minio-init api worker admin client

deploy-ps:
	@$(COMPOSE) ps

deploy-logs:
	@$(COMPOSE) logs -f --tail=200 client admin api worker

deploy-restart:
	@$(COMPOSE) restart client admin api worker

deploy-down:
	@$(COMPOSE) down
