set dotenv-load := true

gen:
    # preprocess the open-api spec
    bash spec/preprocess_swagger.sh
    # generate go client code from the processed spec
    oapi-codegen -package youscore spec/swagger_processed.json > client.gen.go

test: gen
    # check builds
    go build ./...
    # run tests
    go test ./...

download:
    # download latest version of the open-api spec
    curl "https://api.youscore.com.ua/swagger/v1/swagger.json" -o spec/swagger.json

install:
    # install code gen dependencies
    go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

example:
    go run example/main.go