# Deploy-IO

A Simple Vercel Clone's Backend

## Description

Deploy-IO is a backend system designed to replicate the core functionalities of Vercel.

## Installation

Before proceeding, ensure that the following components are installed on your system:

- **Docker**
- **Docker Compose**
- **Golang** (v1.22 or greater)
- **Go Migrate**

To manage the Docker containers, use the following commands:

- Start the containers: `./containers.sh up`
- Stop the containers: `./containers.sh down`

> **Note:** These commands were tested on Linux. If you are using Windows, manually browse the `containers.sh` file and run the commands that start with `docker-compose` to achieve the same functionality.

## Usage

- Every folder except `analysis` contains a `.env.sample` file. Copy it as `.env` and fill it with the necessary values before starting development.
  
- Run migrations one by one from the migrations directory using
  ```bash
  go run migrations.go 000001_init.[up|down].sql [up|down]
  ```

- Start the services using:

  ```bash
  go run .
  ```

- For production environments, avoid using `.env` files. Instead, hardcode the secrets into the operating system's environment variables and start each service with the `-env=prod` flag.

## Features

- Deploy applications that can output static assets
- A grafana based dashboard to monitor servers and files being served