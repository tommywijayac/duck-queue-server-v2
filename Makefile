-include .env

.PHONY: build

# build ubuntu
build:
	@echo " > Building [queue]..."
	@go build -o ./bin/queue .
	@echo " > Finished building [queue]"

run: build
	@echo " > Running [queue]..."
	@./bin/queue
	@echo " > Finished running [queue]"