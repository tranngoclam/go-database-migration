up:
	@docker compose up -d --build

down:
	@docker compose down

ps:
	@docker compose ps

exec:
	@docker exec -it mysql mysql -uroot -ppassword

test:
	@go test ./... -v
