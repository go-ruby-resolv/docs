<!-- SPDX-License-Identifier: BSD-3-Clause -->
# `go-ruby-resolv` library-level benchmark harness

Reproducible, cross-runtime benchmark of the **pure-Go `go-ruby-resolv`
library** against the reference Ruby runtimes (MRI, MRI + YJIT, JRuby,
TruffleRuby). It measures each **offline, pure-compute** `resolv` primitive
through the Go API, isolated from the rbgo interpreter, so the numbers answer:
*is the pure-Go `resolv` as fast as the reference runtime's own stdlib `resolv`
— and does it beat MRI + YJIT?*

## Scope: pure compute, no network

`rbgo`'s `resolv` binding is **compute primitives only — sockets stay stubbed**,
so this harness deliberately benchmarks only the deterministic, offline parts of
`resolv` that need no DNS server: DNS **name** parse/encode, **IPv4/IPv6**
address parse and render, and DNS **message** pack/unpack over a fixed byte
buffer. No socket is opened and no query is sent, so every op is reproducible.

## Layout

- `go/`            — self-contained Go driver; `go.mod` pins the **published**
  library by pseudo-version (no `replace`).
- `ruby/resolv.rb` — the equivalent workload; `ruby/_harness.rb` is the shared
  timer.
- `run.sh`         — runs every available runtime and prints one Markdown table
  per operation (ns/op + ratio vs MRI).

## Run

```sh
bash benchmarks/run.sh
```

Environment knobs: `OUTER` (timed passes, default 25), `WARM` (untimed warm-up
passes, default 3), and `RUBY`/`JRUBY`/`TRUFFLERUBY` to select runtime binaries.

## Method

Each process runs `WARM` untimed passes (to let the JVM/GraalVM JITs warm up),
then `OUTER` timed passes of a fixed inner loop, timed with a monotonic clock;
the **best** pass is reported as **ns/op**. Interpreter start-up is outside the
timed region. The Go driver and the Ruby script build the **identical**
deterministic corpus, and each op's integer checksum is verified identical
across all runtimes (`CHECK=1 go run .` / `CHECK=1 ruby ruby/resolv.rb`) before
timing — including `message-encode`, whose checksum is the byte sum of the
encoded wire, so it only matches if every RDATA encoder and name-compression
pointer agrees. Published, dated results are in
[`../docs/performance.md`](../docs/performance.md).
