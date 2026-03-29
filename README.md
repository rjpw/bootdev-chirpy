# Student Boot.dev Repo: Learn HTTP Servers in Go

This repository contains my implementation of the example server defined in the [Boot.dev](https://www.boot.dev) course [Learn HTTP Servers in Go](https://www.boot.dev/courses/learn-http-servers-golang).

## Progress

This is early days. I won't be pushing commits for every lesson, just when I have important milestones.

## Quickstart for Developers

I use Air while working on features. This provides hot reload of the back end, and saves you from having to stop and start the server all the time.

I'm also using `make` to orchestrate static analysis and testing, with `golangci-lint` to keep me honest.


```bash
# install golangci-lint (optional but a good idea)
curl -sSfL https://golangci-lint.run/install.sh | \
  sh -s -- -b $(go env GOPATH)/bin v2.11.4

# configure air (with any changes you see fit to make)
cp .air-example.toml .air.toml

# run in development mode
make run

# in another terminal
curl localhost:8080
```

