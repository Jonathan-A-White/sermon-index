# Changelog

Newest first, one dated line per change a user of the service or its files could notice; a breaking change says so first.

- 2026-11-05: `sermons serve`, the read-only HTTP API `/v1` (docs/api.md): `/v1/health`, `/v1/sources`, `/v1/sermons?ref=&source=&limit=&offset=` (preaching text first, narrower first), `/v1/themes?ref=`, `/v1/search?q=`; CORS for Lampas's origins, ETag and five-minute caching, gzip, 60 requests a minute per client IP, a log line a request. Migration `0002_search` adds full-text search columns; run `sermons migrate`. `make build` stamps the version.
- 2026-11-05: The first version: the `sermons` binary with `migrate`, `check` and `load`; schema v1 (sources, sermons and their passages as verse-ordinal ranges with a GiST index, themes, theme parts and theme links); the source-directory format (`source.toml` with the licence verbatim, `rows.jsonl`, `topics.jsonl`, `links.jsonl`); a template and a twenty-sermon Spurgeon fixture.
