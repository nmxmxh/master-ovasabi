# Docker Image Optimization Strategies

This document summarizes actionable strategies for reducing Docker image size, inspired by
[Still Shipping 1GB Docker Images? Here's How to Crush Them in Half an Hour](https://medium.com/devlink-tips/still-shipping-1gb-docker-images-heres-how-to-crush-them-in-half-an-hour-e04350ab91f3).

## Why Optimize Docker Images?

- **Faster CI/CD pipelines**
- **Lower cloud costs (e.g., Fargate, GKE, EKS, etc.)**
- **Reduced attack surface and supply chain risk**
- **Faster deployments and rollbacks**

## Quick Wins

- **Use multi-stage builds**: Only copy the final binary and minimal assets into the final image.
- **Start from a minimal base image**: Use `scratch`, `alpine`, or a distroless image for Go apps.
- **Clean up build dependencies**: Remove compilers, package managers, and temp files in the final
  image.
- **.dockerignore**: Exclude unnecessary files (e.g., `.git`, `node_modules`, docs) from the build
  context.

## Advanced Tricks

- **Use tools like [Dive](https://github.com/wagoodman/dive)** to analyze image layers and spot
  bloat.
- **Try [DockerSlim](https://dockersl.im/)** to automatically minimize images.
- **Minimize layers**: Combine `RUN` commands where possible.
- **Pin versions**: Use specific tags for base images and dependencies to ensure reproducibility.

## Example: Go App Multi-Stage Dockerfile

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o master-ovasabi ./cmd/server

FROM alpine:3.19
WORKDIR /app
COPY --from=builder /app/master-ovasabi ./master-ovasabi
COPY --from=builder /app/config ./config
COPY --from=builder /app/amadeus ./amadeus
COPY --from=builder /app/site ./site
COPY --from=builder /app/docs ./docs
CMD ["./master-ovasabi"]
```

## Tools to Use

- [Dive](https://github.com/wagoodman/dive): Visualize and analyze image layers.
- [DockerSlim](https://dockersl.im/): Automatically minify images.
- [Hadolint](https://github.com/hadolint/hadolint): Lint Dockerfiles for best practices.

## When to Optimize

- **Focus on image optimization after the app is stable and the Docker build is reliable.**
- Use this guide as a checklist for future improvements.

## References

- [Still Shipping 1GB Docker Images? Here's How to Crush Them in Half an Hour](https://medium.com/devlink-tips/still-shipping-1gb-docker-images-heres-how-to-crush-them-in-half-an-hour-e04350ab91f3)
