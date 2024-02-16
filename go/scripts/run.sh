#!/bin/sh

GREEN='\033[0;32m'
NC='\033[0m'

nodemon --delay 1s -e go,html,yaml --signal SIGTERM --quiet --exec \
'echo -e "\n'"$GREEN"'[Restarting]'"$NC"'" && go run -ldflags "-X main.version="$(cat VERSION)"" ./go/cmd/rpdb --config ./go/config.yaml' -- "$@" || exit 1""
