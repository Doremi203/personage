secrets:
	@printf '%s\n' \
	"YC_TOKEN=$$(yc iam create-token --impersonate-service-account-id ajemh7rheo5jrvfq4svj)" \
	"YC_CLOUD_ID=$$(yc config get cloud-id)" "YC_FOLDER_ID=$$(yc config get folder-id)" \
	"AWS_ACCESS_KEY_ID=$$(yc lockbox payload get --id e6q869h32umj7dap12qa --key access_key_id)" \
	"AWS_SECRET_ACCESS_KEY=$$(yc lockbox payload get --id e6q869h32umj7dap12qa --key secret_key)" \
	> secrets.env

# Frontend commands
frontend-build:
	cd frontend && npm run build

frontend-deploy:
	@if [ ! -f secrets.env ]; then echo "Run 'make secrets' first"; exit 1; fi
	@set -a && source secrets.env && set +a && cd frontend && ./deploy.sh

frontend-release: frontend-build frontend-deploy

