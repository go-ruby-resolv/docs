# Performance

`go-ruby-resolv/resolv` is the pure-Go, CGO-free library that
[`rbgo`](https://github.com/go-embedded-ruby/ruby) binds for Ruby's `resolv`.
This page records a **real, library-level** benchmark of that module's Go API
against every reference runtime's own stdlib `resolv`, one row per offline
`resolv` primitive. It is part of the ecosystem-wide per-module parity suite, and
**the bar is beating MRI + YJIT**, not just plain MRI.

## Scope: pure compute, no network

`rbgo`'s `resolv` binding is **compute primitives only — the sockets stay
stubbed**. So this benchmark deliberately measures only the deterministic,
**offline** parts of `resolv` that need no DNS server, no socket, and no live
query: DNS **name** parse/encode, **IPv4/IPv6** address parse and render, and DNS
**message** pack/unpack over a fixed byte buffer. Nothing here touches the
network, which is exactly what makes every number reproducible. Live resolution
(`getaddress`, UDP/TCP round-trips) is out of scope by construction.

## What is measured

Eight representative offline operations run over one **fixed, deterministic
corpus** (16 DNS names, 16 IPv4 addresses, 12 IPv6 addresses, and one
representative DNS response message with records spread across the answer,
authority and additional sections):

| Op | What it exercises |
| --- | --- |
| `name-parse` | `Resolv::DNS::Name.create` over all 16 names (label splitting, absolute flag) |
| `name-to_s` | `Resolv::DNS::Name#to_s` rendering of the pre-parsed names |
| `ipv4-parse` | `Resolv::IPv4.create` over all 16 dotted-quads (anchored octet regex) |
| `ipv4-to_s` | `Resolv::IPv4#to_s` rendering of the pre-parsed addresses |
| `ipv6-parse` | `Resolv::IPv6.create` over all 12 textual forms (8-hex, `::`, embedded-IPv4) |
| `ipv6-to_s` | `Resolv::IPv6#to_s` rendering, including first-run zero compression |
| `message-encode` | `Resolv::DNS::Message#encode`: pack the whole message to wire, with 0xC0 name compression |
| `message-decode` | `Resolv::DNS::Message.decode`: unpack the fixed wire buffer back into typed records |

The **go-ruby** column drives this pure-Go library through its Go API; every
other column is that interpreter's own stdlib `resolv`. The Go and Ruby drivers
build the **identical** corpus and, before any timing, each op's integer
checksum is verified **identical across all four runtimes and the Go driver**
(e.g. `name-parse`=72 labels, `ipv6-parse`=5856, `message-decode`=714751). The
`message-encode` checksum is the **byte sum of the encoded wire** (22432), so it
only matches if every RDATA encoder and every name-compression pointer agrees —
making it a strong wire-format equality proof. So the comparison is the same
observable operation, apples-to-apples.

- **Host:** Apple M4 Max, macOS (`arm64-darwin`). **Date:** 2026-07-03.
- **Runtimes:** Go 1.26.4; `ruby 4.0.5 +PRISM` (MRI, the oracle) and
  `ruby --yjit`; `jruby 10.1.0.0` (OpenJDK 25); `truffleruby 34.0.1`
  (GraalVM CE Native).
- **Method:** each process runs 3 untimed warm-up passes then 25 timed passes of
  a fixed inner loop, timed with a monotonic clock; the **best** pass is reported
  as **ns/op**. Interpreter start-up is outside the timed region, so the number
  is the operation's own cost, not `ruby file.rb` process cost. Numbers were
  stable to within a few percent across repeated runs; treat the small
  sub-microsecond gaps as approximate.
- Harness and drivers live in this repo under
  [`benchmarks/`](https://github.com/go-ruby-resolv/docs/tree/main/benchmarks)
  (`go/`, `ruby/resolv.rb`, `run.sh`). Reproduce: `bash benchmarks/run.sh`.

## Results (ns/op, best of 25)

| Op | go-ruby (pure Go) | MRI | MRI + YJIT | JRuby | TruffleRuby | **go vs YJIT** |
| --- | ---: | ---: | ---: | ---: | ---: | ---: |
| `message-decode` | **2 477** | 75 950 | 50 350 | 60 024 | 826 876 | **20.3× faster** ✅ |
| `name-parse` | **1 494** | 33 910 | 27 190 | 22 644 | 32 880 | **18.2× faster** ✅ |
| `message-encode` | **2 934** | 62 210 | 50 840 | 50 940 | 67 757 | **17.3× faster** ✅ |
| `name-to_s` | **800** | 8 080 | 7 380 | 9 520 | 10 642 | **9.2× faster** ✅ |
| `ipv4-parse` | **2 940** | 12 710 | 10 360 | 12 238 | 34 561 | **3.5× faster** ✅ |
| `ipv6-parse` | **9 205** | 27 960 | 25 440 | 61 083 | 48 953 | **2.8× faster** ✅ |
| `ipv4-to_s` | **1 390** | 4 300 | 3 770 | 9 131 | 8 146 | **2.7× faster** ✅ |
| `ipv6-to_s` | **4 906** | 11 300 | 10 780 | 36 555 | 26 280 | **2.2× faster** ✅ |

## The go-vs-YJIT verdict, per op

**The pure-Go library beats MRI + YJIT on every one of the eight offline
operations** — there is no op where YJIT wins. The margins split into two bands:

**Structural pack/parse — 17×–20× faster than YJIT:**

- **`message-decode` — 20.3× faster** (2 477 ns vs 50 350 ns).
- **`name-parse` — 18.2× faster** (1 494 ns vs 27 190 ns).
- **`message-encode` — 17.3× faster** (2 934 ns vs 50 840 ns).

These are the object-heavy operations. In MRI they allocate a tree of `Name`,
`Label` and `Resource` objects and thread them through the encoder/decoder with
per-byte method dispatch; YJIT removes some interpreter overhead but not the
allocation and object-model cost. The Go port does the same work over slices and
structs with no interpreter in the loop, so it pulls far ahead — and this is the
exact shape of the payload `rbgo` binds.

**Regex-bounded address work — 2.2×–3.5× faster than YJIT:**

- **`ipv4-parse` — 3.5× faster**, **`ipv6-parse` — 2.8× faster**.
- **`ipv4-to_s` — 2.7× faster**, **`ipv6-to_s` — 2.2× faster**.

Address parsing is dominated by the anchored `Regex256` / IPv6 alternation match
that both implementations run; the Go side delegates to the sibling pure-Go
Onigmo engine ([`go-ruby-regexp`](https://github.com/go-ruby-regexp/regexp)),
which is why the margin here is narrower than for the allocation-bound ops rather
than another order of magnitude. Rendering (`to_s`) is a small formatting loop
and tracks the same band. Closing the remaining regexp gap further is tracked in
`go-ruby-regexp` and is the top lever for the address ops; even so, the pure-Go
primitive already beats every interpreter, YJIT included, on all four.

## Caveats

- **Cold-JIT framing.** JRuby and TruffleRuby are timed after the same 3 warm-up
  passes as everyone else, but 3 passes do **not** bring the JVM/GraalVM JITs to
  full steady state; read their columns as lightly-warmed, not as peak
  throughput. TruffleRuby's `message-decode` in particular is still deep in its
  warm-up here. MRI, YJIT and Go reach their representative speed almost
  immediately, so their columns are the load-bearing comparison.
- **Offline scope.** These are `resolv`'s compute primitives only; live DNS
  resolution is not measured (and is stubbed in the `rbgo` binding), so this page
  makes no claim about network resolution performance.
- No number here is fabricated: all figures are measured on the host and date
  named above and reproduce with `bash benchmarks/run.sh`.
