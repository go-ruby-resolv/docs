# Roadmap

`go-ruby-resolv/resolv` is grown **test-first**, each capability differential-tested against
MRI rather than built in isolation. Ruby's Resolv — the deterministic,
interpreter-independent slice extracted from rbgo's internals — is **complete**.

| Stage | What | Status |
| --- | --- | --- |
| Resolv::DNS::Message | Full header + question + answer / authority / additional sections; `Encode` / `Decode` round-trip byte-for-byte including the decode-side truncation short-circuit. | **Done** |
| Resolv::DNS::Name | Dotted parse / print, absolute flag, case-insensitive `Equal` / `SubdomainOf`, RFC 1035 labels and `0xC0` compression pointers with backward-pointer and 255-octet guards. | **Done** |
| Resource records | `A`, `AAAA`, `CNAME`, `NS`, `PTR`, `MX`, `TXT`, `SOA`, `SRV`, `HINFO`, plus an opaque `Generic` fallback for any other TYPE/CLASS. | **Done** |
| Resolv::IPv4 / IPv6 | `Create` parse against MRI's exact acceptance set, canonical rendering with first-run `::` compression, raw `Addr`, `Equal`, exported regexes. | **Done** |
| Resolv::Hosts | Parse hosts-file text into name↔address tables with `GetAddress` / `GetName` and the reversed per-name ordering MRI produces. | **Done** |
| Differential oracle & coverage | A wide corpus encoded here and by the system `ruby`, compared byte-for-byte; 100% coverage, gofmt + go vet clean, green across all six 64-bit Go arches and three OSes. | **Done** |

## Documented out-of-scope boundaries

These are **deliberate**, recorded so the module's surface is unambiguous:

- **No networking, no file I/O.** The library is pure compute: message / name / record encode-decode, the address grammar, and the hosts-file *content* parse. The actual resolution — querying a server over UDP/TCP, reading `/etc/hosts` from disk — is the host's job; `rbgo` wires sockets to these primitives.
- **No interpreter.** It never runs arbitrary Ruby; `Resolv.getaddress` and anything that hits the network is out of scope and belongs in the consumer.
- **Reference is reference Ruby (MRI).** Byte-for-byte conformance targets MRI's `Resolv`, pinned by the differential oracle.
- **Standalone & reusable.** The module has no dependency on the Ruby runtime; the dependency runs the other way.

See [Usage & API](api.md) for the surface and [Why pure Go](why.md) for the
deterministic/interpreter split.
