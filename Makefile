run_bot_local:
	docker compose --env-file config/.env.dev -f docker-compose.local.yaml up

run_bundler_production:
	docker compose --env-file config/.env.staging up

run_bundler_production_daemon:
	docker compose --env-file config/.env.staging up -d