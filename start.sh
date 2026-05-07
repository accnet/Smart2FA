#!/bin/bash

# Default port
PORT=8083
APP_NAME="smart2fa"

usage() {
    echo "Usage: $0 [dev|prod]"
    echo "  dev  : Run with 'go run .'"
    echo "  prod : Build binary and run"
    exit 1
}

if [ -z "$1" ]; then
    usage
fi

case "$1" in
    dev)
        echo "Starting in DEV mode on port $PORT..."
        PORT=$PORT go run .
        ;;
    prod)
        echo "Building binary..."
        go build -o $APP_NAME .
        if [ $? -eq 0 ]; then
            echo "Starting in PROD mode on port $PORT..."
            PORT=$PORT ./$APP_NAME
        else
            echo "Build failed!"
            exit 1
        fi
        ;;
    *)
        usage
        ;;
esac
