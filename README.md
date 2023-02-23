![axiom-honeycomb-proxy: Ship logs to Axiom and Honeycomb simultaneously](.github/images/banner-dark.svg#gh-dark-mode-only)
![axiom-honeycomb-proxy: Ship logs to Axiom and Honeycomb simultaneously](.github/images/banner-light.svg#gh-light-mode-only)

<div align="center">

[![Documentation][docs_badge]][docs]
[![Go Workflow][workflow_badge]][workflow]
[![Latest Release][release_badge]][release]
[![License][license_badge]][license]

</div>

[Axiom](https://axiom.co) unlocks observability at any scale.

- **Ingest with ease, store without limits:** Axiom's next-generation datastore
  enables ingesting petabytes of data with ultimate efficiency. Ship logs from
  Kubernetes, AWS, Azure, Google Cloud, DigitalOcean, Nomad, and others.
- **Query everything, all the time:** Whether DevOps, SecOps, or EverythingOps,
  query all your data no matter its age. No provisioning, no moving data from
  cold/archive to "hot", and no worrying about slow queries. All your data, all.
  the. time.
- **Powerful dashboards, for continuous observability:** Build dashboards to
  collect related queries and present information that's quick and easy to
  digest for you and your team. Dashboards can be kept private or shared with
  others, and are the perfect way to bring together data from different sources.

For more information check out the
[official documentation](https://axiom.co/docs) and our
[community Discord](https://axiom.co/discord).

## Usage

There are multiple ways you can install _Axiom Honeycomb Proxy_:

- With Homebrew: `brew install axiomhq/tap/axiom`
- Download the pre-built binary from the
  [GitHub Releases](https://github.com/axiomhq/axiom-honeycomb-proxy/releases/latest)
- Using Go:
  `go install github.com/axiomhq/axiom-honeycomb-proxy/cmd/axiom@latest`
- Use the
  [Docker image](https://hub.docker.com/r/axiomhq/axiom-honeycomb-proxy):
  `docker run axiomhq/axiom-honeycomb-proxy`

Create an api token in `Settings > API Tokens` with minimal privileges (`ingest`
permission for the dataset(s) you want to ingest into) and export it as
`AXIOM_TOKEN`.

Alternatively, if you use the [Axiom CLI](https://github.com/axiomhq/cli), run
`eval $(axiom config export -f)` to configure your environment variables.
Otherwise create a personal token in
[the Axiom settings](https://app.axiom.co/profile) and export it as
`AXIOM_TOKEN`. Set `AXIOM_ORG_ID` to the organization ID from the settings
page of the organization you want to access.

Run it: `axiom-honeycomb-proxy` or using Docker:

```shell
docker run -p8080:8080/tcp \
  -e=AXIOM_TOKEN=<YOUR_AXIOM_TOKEN> \
  axiomhq/axiom-honeycomb-proxy
```

**Important:** Honeycomb creates datasets when you push data to them. Axiom does
not support this. Make sure you create the matching datasets in Axiom, first.

Point all Honeycomb related tools at the proxy deployment which accepts data on
the following endpoints:

### Single event requests

```shell
curl http://localhost:8080/honeycomb/v1/events/<DATASET> -X POST \
  -H "X-Honeycomb-Team: <YOUR-HONEYCOMB-KEY>" \
  -H "X-Honeycomb-Event-Time: 2018-02-09T02:01:23.115Z" \
  -d '{"method":"GET","endpoint":"/foo","shard":"users","dur_ms":32}'
```

### Event batch requests

```shell
curl  http://localhost:8080/honeycomb/v1/batch/<DATASET> -X POST \
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

## License

Distributed under the [MIT License](./LICENSE).

<!-- Badges -->

[docs]: https://docs.axiom.co
[docs_badge]: https://img.shields.io/badge/docs-reference-blue.svg
[workflow]: https://github.com/axiomhq/axiom-honeycomb-proxy/actions/workflows/push.yaml
[workflow_badge]: https://img.shields.io/github/actions/workflow/status/axiomhq/axiom-honeycomb-proxy/push.yaml?branch=main&ghcache=unused
[release]: https://github.com/axiomhq/axiom-honeycomb-proxy/releases/latest
[release_badge]: https://img.shields.io/github/release/axiomhq/axiom-honeycomb-proxy.svg
[license]: https://opensource.org/licenses/MIT
[license_badge]: https://img.shields.io/github/license/axiomhq/axiom-honeycomb-proxy.svg?color=blue
