.PHONY: bump
bump:
	@echo "🚀 Bumping Version"
	git tag $(shell svu patch)
	git push --tags

.PHONY: build
build:
	@echo "🚀 Building Version $(shell svu current)"
	go build -o imgcat ./cmd/imgcat/main.go

.PHONY: test
test:
	@echo "🧪 Running Tests"
	go test ./... -v

.PHONY: test-short
test-short:
	@echo "🧪 Running Tests (Short Mode)"
	go test ./... -short

.PHONY: test-race
test-race:
	@echo "🧪 Running Tests with Race Detection"
	go test ./... -race

.PHONY: demo
demo:
	@echo "🚀 Running Demo"
	go run ./cmd/demo/main.go

PROTOCOL ?=
FORCE_TMUX ?=

.PHONY: debug
debug:
	@echo "🚀 Starting dlv debug server on :2345"
	@echo "VSCode debugger can now attach to localhost:2345"
	dlv debug --headless --listen=:2345 --api-version=2 ./cmd/imgcat/main.go -- -P $(PROTOCOL) $(FORCE_TMUX) --clear --dither -W 200 -H 151 ./test/image_smol.png

.PHONY: debug-auto
debug-auto: PROTOCOL := auto
debug-auto: debug

.PHONY: debug-kitty
debug-kitty: PROTOCOL := kitty
debug-kitty: debug

.PHONY: debug-iterm
debug-iterm: PROTOCOL := iterm
debug-iterm: debug

.PHONY: debug-iterm
debug-iterm: PROTOCOL := iterm FORCE_TMUX := --tmux
debug-iterm: debug

.PHONY: debug-sixel
debug-sixel: PROTOCOL := sixel
debug-sixel: debug

.PHONY: debug-halfblocks
debug-halfblocks: PROTOCOL := halfblocks
debug-halfblocks: debug