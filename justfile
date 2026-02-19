set dotenv-load := true

install-deps:
    # install code gen dependencies
    go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest

download-spec:
    # download latest version of the open-api spec
    curl "https://api.youscore.com.ua/swagger/v1/swagger.json" -o spec/swagger.json

translate:
    # translate swagger to english
    go run spec/translate_swagger.go

    # validate changed lines (excluding comments)
    ./spec/verify_translate_diff.sh

gen:
    # preprocess the open-api spec
    bash spec/preprocess_swagger.sh

    # generate go client code from the processed spec
    oapi-codegen -package youscore spec/swagger_en_processed.json > client.gen.go
    go mod tidy

test: gen
    # check builds
    go build ./...

    # run tests
    go test ./...

example:
    go run example/main.go