#!/bin/bash

if [ $# -eq 0 ]; then
    echo "Specify whether to up or down the containers."
    exit 1
fi

arg1=$1

if [ "$arg1" == "up" ]; then
    echo "Starting the services..."
    docker-compose -f docker-data.yml up -d
    docker-compose -f ./analysis/prom-graf.yml up -d

elif [ "$arg1" == "down" ]; then
    echo "Winding up the services..."
    docker-compose -f docker-data.yml down
    docker-compose -f ./analysis/prom-graf.yml down

else
    echo "Invalid option: $arg1"
    exit 1
fi