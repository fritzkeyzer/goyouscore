#!/bin/bash
# Preprocesses swagger.json to fix parameter names that conflict with Go package names.
# The parameter "url" shadows Go's "net/url" package in generated code.

sed -e 's|{url}|{photoUrl}|g' \
    -e 's|"name": "url"|"name": "photoUrl"|g' \
    spec/swagger.json > spec/swagger_processed.json
