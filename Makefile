all:
	go test ./...
	gofmt -w -s `go list -f '{{.Dir}}' ./...`
	go build -o z42-resolver ./cmd/resolver
	go build -o z42-api ./cmd/api
	go build -o z42-updater ./cmd/zone_updater
	go build -o z42-healthchecker ./cmd/healthchecker

test:
	go test -v ./...

clean:
	rm -f z42-resolver z42-api z42-updater z42-healthchecker
