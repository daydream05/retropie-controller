APP := retropie-controller

.PHONY: build build-pi run

build:
	go build -o $(APP) .

build-pi:
	GOOS=linux GOARCH=arm GOARM=7 go build -o $(APP)-arm .

run:
	go run .
