# Axiom Honeycomb Proxy

[![Go Workflow][go_workflow_badge]][go_workflow]
[![Coverage Status][coverage_badge]][coverage]
[![Go Report][report_badge]][report]
[![Latest Release][release_badge]][release]
[![License][license_badge]][license]
[![Docker][docker_badge]][docker]

---

## Table of Contents

1. [Introduction](#introduction)
1. [Usage](#usage)
1. [Contributing](#contributing)
1. [License](#license)

## Introduction

_Axiom Honeycomb Proxy_ ships logs to Axiom and Honeycomb simultaneously.

## Installation

### Download the pre-compiled and archived binary manually

Binary releases are available on [GitHub Releases][2].

  [2]: https://github.com/axiomhq/axiom-honeycomb-proxy/releases/latest

### Install using [Homebrew](https://brew.sh)

```shell
brew tap axiomhq/tap
brew install axiom-honeycomb-proxy
```

To update:

```shell
brew update
brew upgrade axiom-honeycomb-proxy
```

### Install using `go get`

```shell
go get -u github.com/axiomhq/axiom-honeycomb-proxy/cmd/axiom-honeycomb-proxy
```

### Install from source

```shell
git clone https://github.com/axiomhq/axiom-honeycomb-proxy.git
cd axiom-honeycomb-proxy
make install
```

### Run the Docker image

Docker images are available on [DockerHub][docker].

## Usage

1. Set the following environment variables:

* `AXIOM_URL`: **https://cloud.axiom.co**
* `AXIOM_TOKEN`: **Personal Access** or **Ingest** token. Can be
created under `Profile` or `Settings > Ingest Tokens`. For security reasons it
is advised to use an Ingest Token with minimal privileges only.

1. Run it: `./axiom-honeycomb-proxy` or using docker:

```shell
docker run -p8080:8080/tcp \
  -e=AXIOM_URL=<https://cloud.axiom.co> \
  -e=AXIOM_TOKEN=<xapt-xxxxx-xxxxxxx> \
  axiomhq/axiom-honeycomb-proxy
```

3. Point all Honeycomb related tools at the proxy deployment.

### Request format

#### Single event requests

```shell
curl http://localhost:3111/honeycomb/v1/events/<DATASET> -X POST \
  -H "X-Honeycomb-Team: <YOUR-HONEYCOMB-KEY>" \
  -H "X-Honeycomb-Event-Time: 2018-02-09T02:01:23.115Z" \
  -d '{"method":"GET","endpoint":"/foo","shard":"users","dur_ms":32}'
```

#### Event batch requests

```shell
curl  http://localhost:3111/honeycomb/v1/batch/<DATASET> -X POST \
  -H "X-Honeycomb-Team: <YOUR-HONEYCOMB-KEY>" \
  -d '[
        {
          "time":"2018-02-09T02:01:23.115Z",
          "data":{"key1":"val1","key2":"val2"}
        },
        {
          "data":{"key3":"val3"}
        }
      ]'
```

### Note

Honeycomb creates datasets when you push data to them. Axiom does not support
that (yet). Make sure you create the matching datasets on the Axiom side, first.

## Contributing

Feel free to submit PRs or to fill issues. Every kind of help is appreciated. 

Before committing, `make` should run without any issues.

Kindly check our [Contributing](Contributing.md) guide on how to propose
bugfixes and improvements, and submitting pull requests to the project.

## License

&copy; Axiom, Inc., 2021

Distributed under MIT License (`The MIT License`).

See [LICENSE](LICENSE) for more information.

<!-- Badges -->

[go_workflow]: https://github.com/axiomhq/axiom-honeycomb-proxy/actions/workflows/push.yml
[go_workflow_badge]: https://img.shields.io/github/workflow/status/axiomhq/axiom-honeycomb-proxy/Push?style=flat-square&ghcache=unused
[coverage]: https://codecov.io/gh/axiomhq/axiom-honeycomb-proxy
[coverage_badge]: https://img.shields.io/codecov/c/github/axiomhq/axiom-honeycomb-proxy.svg?style=flat-square&ghcache=unused
[report]: https://goreportcard.com/report/github.com/axiomhq/axiom-honeycomb-proxy
[report_badge]: https://goreportcard.com/badge/github.com/axiomhq/axiom-honeycomb-proxy?style=flat-square&ghcache=unused
[release]: https://github.com/axiomhq/axiom-honeycomb-proxy/releases/latest
[release_badge]: https://img.shields.io/github/release/axiomhq/axiom-honeycomb-proxy.svg?style=flat-square&ghcache=unused
[license]: https://opensource.org/licenses/MIT
[license_badge]: https://img.shields.io/github/license/axiomhq/axiom-honeycomb-proxy.svg?color=blue&style=flat-square&ghcache=unused
[docker]: https://hub.docker.com/r/axiomhq/axiom-honeycomb-proxy
[docker_badge]: https://img.shields.io/docker/pulls/axiomhq/axiom-honeycomb-proxy.svg?style=flat-square&ghcache=unused