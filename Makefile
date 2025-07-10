# Переменные
GETH_URL = http://localhost:8545
CONTRACT_REPO = https://github.com/cyberkotletki/contracts.git
CONTRACT_REPO_RAW = https://raw.githubusercontent.com/cyberkotletki/contracts/main/contracts
CONTRACT_DIR = ./contracts
ABI_DIR = ./internal/abi
SOLC_VERSION = 0.8.19
CONTRACTS = Donates.sol Types.sol

# Цели по умолчанию
.PHONY: contracts-clone contracts-compile contracts-deploy geth-accounts dev-setup install-tools

# Клонирование только папки contracts из репозитория
contracts-clone:
	@echo "Клонирование папки contracts из репозитория..."
	@if [ -d "$(CONTRACT_DIR)" ]; then \
		echo "Директория контрактов уже существует, удаляем и создаём заново..."; \
		rm -rf $(CONTRACT_DIR); \
	fi
	@mkdir -p $(CONTRACT_DIR)
	@for contract in $(CONTRACTS); do \
		echo "Скачиваем $$contract..."; \
		curl -fsSL $(CONTRACT_REPO_RAW)/$$contract -o $(CONTRACT_DIR)/$$contract; \
	done
	@echo "Контракты скачаны в $(CONTRACT_DIR)"

# Компиляция контрактов
contracts-compile: contracts-clone ## Компилировать смарт-контракты и генерировать ABI/Go биндинги
	@echo "Компиляция смарт-контрактов..."
	@mkdir -p $(ABI_DIR)
	@if [ -z "$$(ls -A $(CONTRACT_DIR)/*.sol 2>/dev/null)" ]; then \
		echo "Файлы .sol не найдены в $(CONTRACT_DIR)"; \
		echo "Содержимое директории:"; \
		ls -la $(CONTRACT_DIR); \
		exit 1; \
	fi
	@echo "Найденные .sol файлы:"; \
	ls -1 $(CONTRACT_DIR)/*.sol
	@echo "Компиляция с помощью solc..."
	@for contract in $(CONTRACT_DIR)/*.sol; do \
		echo "Компилируем $$contract..."; \
		contract_name=$$(basename $$contract .sol); \
		solc --abi --bin --via-ir --overwrite -o $(ABI_DIR) $$contract; \
		if [ -f "$(ABI_DIR)/$$contract_name.abi" ]; then \
			echo "Генерируем Go биндинги для $$contract_name..."; \
			abigen --abi="$(ABI_DIR)/$$contract_name.abi" \
				--bin="$(ABI_DIR)/$$contract_name.bin" \
				--pkg=contracts \
				--type=$$contract_name \
				--out="$(ABI_DIR)/$$(echo $$contract_name | tr '[:upper:]' '[:lower:]').go"; \
		fi; \
	done
	@echo "ABI файлы и Go биндинги сохранены в $(ABI_DIR)"

# Установка зависимостей
install-tools: ## Установить необходимые утилиты (solc, abigen)
	@echo "Устанавливаем solc..."
	@brew install solidity
	@echo "Устанавливаем abigen..."
	@go install github.com/ethereum/go-ethereum/cmd/abigen@latest
	@echo "Проверяем установку:"
	@solc --version
	@abigen --version

# Создание аккаунтов в geth dev
geth-accounts: ## Создать тестовые аккаунты в geth dev
	@echo "Создание тестовых аккаунтов..."
	@echo "password" | docker exec -i donly-geth-dev geth account new --password /dev/stdin --datadir /root/.ethereum
	@echo "password" | docker exec -i donly-geth-dev geth account new --password /dev/stdin --datadir /root/.ethereum
	@echo "Список аккаунтов:"
	@docker exec donly-geth-dev geth account list --datadir /root/.ethereum

# Деплой контрактов
contracts-deploy: contracts-compile ## Деплоить контракты в локальную geth сеть
	@echo "Деплой контрактов в локальную сеть..."
	@docker run --rm \
		--network donly-network \
		-v $(PWD)/$(CONTRACT_DIR):/contracts \
		-v $(PWD)/deploy-script.js:/deploy.js \
		node:18-alpine \
		sh -c "cd /contracts && npm install web3 && node /deploy.js"

# Проверка статуса geth
geth-status: ## Проверить статус geth ноды
	@echo "Статус geth ноды:"
	@curl -s -X POST -H "Content-Type: application/json" \
		--data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
		$(GETH_URL) | jq '.'

# Получение баланса аккаунта
geth-balances: ## Получить баланс первого аккаунта
	@echo "Баланс первого аккаунта:"
	@docker exec donly-geth-dev geth --exec "eth.accounts.forEach((acc, i) => console.log('Account ' + i + ' (' + acc + '): ' + web3.fromWei(eth.getBalance(acc), 'ether') + ' ETH'))" attach http://localhost:8545

# Настройка среды разработки
dev-setup: install-tools contracts-clone contracts-compile geth-accounts ## Полная настройка среды разработки
	@echo "Среда разработки настроена!"
	@echo "Для деплоя контрактов выполните: make contracts-deploy"

# Очистка
clean: ## Очистить сгенерированные файлы
	@echo "Очистка..."
	@rm -rf $(CONTRACT_DIR) $(ABI_DIR)