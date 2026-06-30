# go-ruby-resolv documentation

**Ruby's Resolv DNS primitives in pure Go — wire format, names, records, addresses, hosts; MRI byte-exact, no cgo.**

`go-ruby-resolv/resolv` is a faithful, pure-Go (zero cgo) reimplementation of Ruby's
[`Resolv`](https://docs.ruby-lang.org/en/master/Resolv.html), matching reference Ruby (MRI) byte-for-byte. The module
path is `github.com/go-ruby-resolv/resolv`.

It was **extracted from rbgo's internals into a reusable standalone library**:
the module is standalone and importable by any Go program, and it is the backend
bound into [go-embedded-ruby](https://github.com/go-embedded-ruby/ruby) by `rbgo`
as a native module — a sibling of [go-ruby-regexp](https://github.com/go-ruby-regexp/regexp) and [go-ruby-marshal](https://github.com/go-ruby-marshal/marshal). The dependency runs the other
way: this library has **no dependency on the Ruby runtime**.

!!! success "Status: complete — MRI byte-exact"
    Messages, names, addresses and hosts tables built here and compared byte-for-byte (base64-framed) against the system `ruby`; 100% coverage, gofmt + go vet clean, green across all six 64-bit Go arches and three OSes.

## Quick taste

```go
// Build and encode a DNS query (Resolv::DNS::Message#encode).
m := resolv.NewMessage(0x1234)
m.AddQuestion(resolv.NewName("www.example.com"), resolv.TypeA, resolv.ClassIN)
fmt.Println(hex.EncodeToString(m.Encode()))

// Decode a response and read its records (Resolv::DNS::Message.decode).
resp, _ := resolv.Decode(m.Encode())
fmt.Println(resp.Question[0].Name) // www.example.com

// Parse addresses (Resolv::IPv4 / Resolv::IPv6).
ip, _ := resolv.CreateIPv6("2001:db8:0:0:0:0:0:1")
fmt.Println(ip)                    // 2001:db8::1

// Parse a hosts table (Resolv::Hosts).
h := resolv.ParseHosts("127.0.0.1 localhost\n")
addr, _ := h.GetAddress("localhost")
fmt.Println(addr)                  // 127.0.0.1
```

## Repositories

| Repo | What it is |
| --- | --- |
| [`resolv`](https://github.com/go-ruby-resolv/resolv) | the library — Ruby's Resolv in pure Go |
| [`docs`](https://github.com/go-ruby-resolv/docs) | this documentation site (MkDocs Material, versioned with mike) |
| [`go-ruby-resolv.github.io`](https://github.com/go-ruby-resolv/go-ruby-resolv.github.io) | the organization landing page (Hugo) |
| [`brand`](https://github.com/go-ruby-resolv/brand) | logo and brand assets |

## Principles

- **Pure Go, `CGO_ENABLED=0`** — trivial cross-compilation, a single static
  binary, no C toolchain.
- **MRI byte-exact.** Output matches reference Ruby exactly, not approximately,
  validated by a differential oracle against the `ruby` binary.
- **Standalone & reusable.** Extracted from rbgo's internals; no dependency on
  the Ruby runtime — the dependency runs the other way.
- **100% test coverage** is the target, enforced as a CI gate.

## Where to go next

- [Why pure Go](why.md) — why this slice of Ruby is deterministic enough to live
  as a standalone, interpreter-independent Go library.
- [Usage & API](api.md) — the public surface and worked examples.
- [Roadmap](roadmap.md) — what is done and what is downstream by design.

Source lives at [github.com/go-ruby-resolv/resolv](https://github.com/go-ruby-resolv/resolv).
