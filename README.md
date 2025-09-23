# INOS – The Internet Native Operating System

> **🚧 Work in Progress (WIP):**  
> INOS is a fast-evolving, open platform for distributed, AI-powered, and WASM-enabled systems. We
> welcome contributors of all backgrounds—Go developers, AI/ML engineers, database and distributed
> systems specialists, QA/testers, frontend and WASM enthusiasts, and anyone passionate about
> building resilient, extensible digital infrastructure. See the Contributing section below to get
> involved!

**INOS** is a self-documenting, AI-ready, and community-driven platform for orchestrating digital services, relationships, and value. It is an open platform for distributed, AI-powered, and WASM-enabled systems.

For a deep dive into the project's vision, architecture, and technical details, please refer to the [WHITE_PAPER.md](WHITE_PAPER.md).

## Getting Started

To get started with INOS, you'll need to set up your environment and run the application.

### Environment Configuration

- Create a `.env` file in the root of the project.
- Use the `.env.example` file as a template for the required environment variables.

### Deployment

The `Makefile` in the root of the project contains all the necessary commands for building, running, and deploying the application.

#### Docker

To build the application using Docker, use the following command:

```bash
make docker-build
```

To run the application using Docker, use the following command:

```bash
make docker-up
```

This will start all the necessary services in Docker containers.

#### WebAssembly (WASM)

To build the WASM module, use the following command:

```bash
make wasm-build
```

This will compile the Go code into a WebAssembly module that can be run in the browser.

For more information on the available commands, you can run `make help`.

## Contributing

We welcome contributors of all backgrounds—Go developers, AI/ML engineers, database and distributed systems specialists, QA/testers, frontend and WASM enthusiasts, and anyone passionate about building resilient, extensible digital infrastructure.

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines. All are welcome—code, docs, ideas, and feedback!

## License

Inos is dual-licensed:

- **MIT License:** Free and open source for community use, contributions, and research. See [LICENSE](LICENSE).
- **Enterprise License (AGPL/BUSL):** For advanced features, enterprise support, and legal guarantees. See [LICENSE](LICENSE).