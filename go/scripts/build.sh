#/bin/sh

go build -o RPdb-go ./cmd/rpdb

# Apply shell completion code
source <(./RPdb-go completion bash)