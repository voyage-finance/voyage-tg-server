# voyage-tg-server

## General info
The code of telegram bot which facilitates the interaction with [Safe Gnosis](https://app.safe.global/).

With this telegram bot you can do:
1. Bind safe vault to chat
2. Bind personal wallets to telegram account. In this way you can use telegram account name instead of wallet addresses
3. Shortcut for create *request transaction* with cli and sing it easily: `/request 100 $usdt`
4. Create stream payment with [Llama pay](https://llamapay.io/)
5. And other features which will help you and/or your organizations to have better UX

To run all flow you have to up/implement:
- up Database, in our case we used postgres
- Retrieve all secrets from external services, like `BOT_API_KEY`, `infura` and etc
- Implement UI to sign messages

## Code Architecture
- `/config` - stores environment files, configs and handles secrets. For further setup you have to look at ./config/README.md
- `/contracts` - package for work with smart contracts. Store any logic in this folder if it interacts with SC
- `http_server` - package to handle separate http server which runs along with telegram-bot code. 
It is needed to handle UI responses. It share the same DB with telegram bot code
- `/models` - store database models
- `/service` - store main reusable services in telegram bot. Like: http client, db client, Safe interaction and etc
- `/transaction` - consist logic of Safe `queue`, `history` and `transaction builder`
- `main.go` - telegram bot entrypoint code where all commands are set


## CI/CD
To run code: `go add ./...` and `go run main.go`. This code will run telegram bot code and http server at port=8070

For CD we used `docker` for different environments(dev, staging, production). You can run:
- `docker build -t voyage_safe_bot .` 
- for up containers you can take a look to `Makefile`
- `docker-compose.local.yaml` - is used for local running of containers, it does not consist nginx
- `docker-compose.yaml` - is used for staging and production
- `/.github` - folder consist CI code with git actions. But you need extra step to up it on server, e.g. nginx, traefik and etc

