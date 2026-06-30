# Why pure Go

`go-ruby-resolv/resolv` reimplements Ruby's Resolv in **pure Go, with cgo disabled**. The
slice of Ruby it covers is **deterministic and interpreter-independent**: given
its inputs, the result is a pure function of those inputs — no live binding, no
evaluation of arbitrary Ruby. That is exactly the part that can — and should —
live as a standalone Go library, separate from the interpreter.

## Extracted from rbgo, reusable by anyone

This library began life inside [go-embedded-ruby](https://github.com/go-embedded-ruby/ruby)'s
`rbgo` — in the internals that back the Ruby surface. It has been **extracted
into a reusable standalone library** so that:

- any Go program can import `github.com/go-ruby-resolv/resolv` directly, with no Ruby runtime;
- the dependency runs the *other* way — `rbgo` binds this module as a native
  module (the same pattern as [go-ruby-regexp](https://github.com/go-ruby-regexp/regexp) and [go-ruby-marshal](https://github.com/go-ruby-marshal/marshal)), rather than this module depending
  on the interpreter;
- the behaviour is pinned by a **differential oracle** against the system
  `ruby`, independent of any one consumer.

## Why pure Go matters here

Because the library is CGO-free and dependency-free, it:

- cross-compiles to every Go target with no C toolchain, and links into a single
  static binary;
- has **no dependency on the Ruby runtime** — the dependency runs the other way;
- can be differentially tested against the `ruby` binary wherever one is on
  `PATH`, while the cross-arch lanes (where `ruby` is absent) still validate the
  library itself.

See [Usage & API](api.md) for the surface and [Roadmap](roadmap.md) for what is
in scope.
