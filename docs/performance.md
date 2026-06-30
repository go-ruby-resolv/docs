# Performance

`go-ruby-resolv/resolv` is the pure-Go library that
[`rbgo`](https://github.com/go-embedded-ruby/ruby) binds for Ruby's `resolv`. This
page records the **methodology** of the ecosystem-wide per-module parity suite —
how this module is benchmarked against the reference Ruby runtimes — without
quoting numbers here, so the figures never drift out of date.

## What is measured

The **same** Ruby script — a DNS encode/decode + address-parse workload (`Resolv::DNS::Message#encode` / `.decode`, `Resolv::IPv4` / `IPv6.create`, `Resolv::Hosts`) — is run under every runtime.
`rbgo`'s number reflects **this pure-Go library doing the work**; every other
column is that interpreter's own `resolv` (or equivalent) implementation. So the
comparison is the **Ruby-visible operation**, apples-to-apples across
interpreters. The script prints a deterministic checksum and its output is
checked **byte-identical to MRI** before timing.

## Method

- **Best-of-N wall time** (best, not mean, to suppress scheduler noise);
  single-shot processes, no warm-up beyond the script's own loop.
- **Runtimes:** MRI (the oracle) and MRI + YJIT; JRuby (OpenJDK); TruffleRuby
  (GraalVM CE Native). JRuby and TruffleRuby are timed **cold, single-shot**, so
  they carry JVM / Graal startup on every run — read them as one-shot
  `ruby file.rb` costs, the same way `rbgo` and MRI are measured, not as
  steady-state JIT numbers.
- The benchmark script and harness live in rbgo's repo under
  [`bench/modules/`](https://github.com/go-embedded-ruby/ruby/tree/main/bench/modules)
  (`resolv.rb` + `run.sh`). Reproduce:
  `RBGO=./rbgo TRUFFLE=truffleruby bash bench/modules/run.sh 5`.

!!! note "Honest framing"
    No headline numbers are reproduced on this page on purpose: the parity suite
    is the source of truth and is re-run per release. Rows that complete in well
    under a couple hundred milliseconds carry the most relative noise; treat
    their ratios as order-of-magnitude. The published figures are real measured
    numbers — nothing is cherry-picked.
