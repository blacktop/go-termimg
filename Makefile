.PHONY: bump
bump:
	@echo "🚀 Bumping Version"
	git tag $(shell svu patch)
	git push --tags

.PHONY: build
build:
	@echo "🚀 Building Version $(shell svu current)"
	go build -o imgcat ./cmd/imgcat/main.go

.PHONY: demo
demo:
	@echo "🚀 Running Demo"
	go run ./cmd/demo/main.go