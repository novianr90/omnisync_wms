.PHONY: up up-local up-full down logs

# Supabase / external database mode
up:
	docker compose up --build -d

# Local PostgreSQL mode
up-local:
	docker compose -f docker-compose.yml -f docker-compose.local-db.yml up --build -d

# Local PostgreSQL + Portainer monitoring + Cloudflare tunnel
up-full:
	docker compose --profile monitoring --profile tunnel \
		-f docker-compose.yml -f docker-compose.local-db.yml up --build -d

down:
	docker compose --profile monitoring --profile tunnel \
		-f docker-compose.yml -f docker-compose.local-db.yml down

logs:
	docker compose logs -f

# Create first admin user (production use)
# Usage: make create-admin EMAIL=admin@example.com PASSWORD=securepassword
create-admin:
	docker compose exec auth ./auth_services create-admin $(EMAIL) $(PASSWORD)
