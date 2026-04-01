build:
	go build -v ./...

test:
	go test ./...

netlify:
	mkdir -p functions && go get ./... && CGO_ENABLED=0 go build -o functions/ ./cmd/cronometer

release:
	mkdir -p bin && go get ./... && CGO_ENABLED=0 go build -o bin/ ./cmd/garmin