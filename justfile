build:
  @echo building for windows...
  GOOS=windows GOARCH=amd64 go build .
  @echo building for linux...
  GOOS=linux GOARCH=amd64 go build .
  @echo building for mac...
  GOOS=darwin GOARCH=amd64 go build -o credder-mac .

build-local:
  go build . && cp credder ~/.local/bin/credder
