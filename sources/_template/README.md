# The template of a source directory

Copy `sources/_template/` to `sources/<id>/` to add a preacher or a collection, and follow [docs/adding-a-source.md](../../docs/adding-a-source.md): it is the recipe, step by step, from the check of the host's terms and `robots.txt` before any crawl or download (the clause quoted on the bead before release) to the credit line. `source.toml` here has every key and every `rows.jsonl` field explained in its comments; `rows.jsonl` has one row with every field, one with only what is required, one with no single text, and one `series` row.

This directory is never loaded: its name begins with `_`, so `sermons load --all` and `sermons check --all` skip it. `sermons check sources/_template` passes, and a test keeps it so.

A source directory holds only `source.toml` and its rows (`rows.jsonl`, or `topics.jsonl` and `links.jsonl` for a theme source). A `README.md` in it, like this one, or a `REPORT.md` of what a tool could not read, is for the person reading the directory and is not read by `sermons`.
