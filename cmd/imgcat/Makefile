.PHONY: debug
debug: ## Debug in another terminal
	@echo " > Debugging"
	@dlv debug --headless --listen=:2345 --api-version=2 main.go -- -V --clear ../../test/image_smol.png

.PHONY: work
work: clean ## Work on go-termimg package
	@echo " > Working on go-termimg package"
	@go work init
	@go work use .  ../../../go-termimg

.PHONY: clean
clean: ## Clean up go.work
	@echo " > 🧼 Cleaning up"
	@rm go.work || true
	@rm go.work.sum || true