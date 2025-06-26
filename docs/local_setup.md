# Local Development Setup Guide

## Prerequisites

- **Docker** and **Docker Compose** installed ([Get Docker](https://docs.docker.com/get-docker/))
- **Make** installed (comes by default on most Unix systems)
- (Optional) **Go** and **Node.js** if you want to run/test outside containers

---

## 1. Clone the Repository

```sh
git clone https://github.com/your-org/master-ovasabi.git
cd master-ovasabi
```

---

## 2. Environment Variables

Copy the example environment file and adjust as needed:

```sh
cp .env.example .env
```

Edit `.env` to set secrets, database credentials, etc.

---

## 3. Build and Start Services

The `Makefile` provides all the main commands. **Always use `make` for setup, build, and test to
ensure consistency!**

### To build and start everything (backend, database, redis, etc):

```sh
make docker-up
```

This will:

- Build all Docker images
- Start all services defined in `docker-compose.yml`

### To stop all services:

```sh
make docker-down
```

---

## 4. Running Migrations

If you need to run database migrations:

```sh
make migrate
```

_(Check the Makefile for the exact migration command if it differs.)_

---

## 5. Running Tests

To run all tests (unit + integration):

```sh
make test
```

For only unit tests:

```sh
make test-unit
```

For integration tests:

```sh
make test-integration
```

---

## 6. Useful Makefile Commands

- **Build the backend:**
  ```sh
  make build
  ```
- **Clean build artifacts:**
  ```sh
  make clean
  ```
- **View logs:**
  ```sh
  make docker-logs
  ```
- **Format and validate docs:**
  ```sh
  make docs-format
  make docs-validate
  ```
- **Lint the codebase:**
  ```sh
  make lint
  ```

---

## 7. Accessing the App

- **API/Backend:**  
  Usually available at `http://localhost:8080` (check `docker-compose.yml` for ports)
- **Docs:**  
  Serve locally with
  ```sh
  make docs-serve
  ```
  Then visit the provided URL (often `http://localhost:8000`).

---

## 8. Stopping and Cleaning Up

To stop all containers and remove volumes:

```sh
make docker-down
make docker-clean
```

---

## 9. Troubleshooting

- If you encounter issues, try rebuilding everything:
  ```sh
  make docker-clean
  make docker-up
  ```
- Check `.env` and `docker-compose.yml` for correct configuration.
- For more commands, run:
  ```sh
  make help
  ```

---

## 10. Contributing

See [CONTRIBUTING.md](../CONTRIBUTING.md) and [README.md](../README.md) for more details on
contributing, coding standards, and project philosophy.

---

**Welcome to Master by Ovasabi!**  
If you have questions, open an issue or ask in the community chat.
