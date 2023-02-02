run_bot_local:
	docker compose --env-file config/.env.dev -f docker-compose.local.yaml up

run_bundler_staging:
	docker compose --env-file config/.env.staging up

run_bundler_staging_daemon:
	docker compose --env-file config/.env.staging down && docker compose --env-file config/.env.staging up -d

run_bundler_production:
	docker compose --env-file config/.env.production up

run_bundler_production_daemon:
	docker compose --env-file config/.env.production down && docker compose --env-file config/.env.production up -d