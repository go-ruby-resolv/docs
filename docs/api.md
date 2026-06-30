# Usage & API

The public API lives at the module root (`github.com/go-ruby-resolv/resolv`). It is **Ruby-shaped but Go-idiomatic**: the types mirror `Resolv::DNS::Message` / `Name` / `IPv4` / `IPv6` / `Hosts`, while the surface follows Go conventions — value types, explicit `error`, no global state, and no networking or file I/O.

!!! success "Status: implemented"
    The library is built and importable as `github.com/go-ruby-resolv/resolv`, bound into
    `rbgo` as a native module; see [Roadmap](roadmap.md).

## Install

```sh
go get github.com/go-ruby-resolv/resolv
```

## Worked example

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

## Shape

```go
// Messages
func NewMessage(id uint16) *Message
func (m *Message) AddQuestion(name Name, typ, class uint16)
func (m *Message) AddAnswer(name Name, ttl uint32, data Resource)
func (m *Message) AddAuthority(name Name, ttl uint32, data Resource)
func (m *Message) AddAdditional(name Name, ttl uint32, data Resource)
func (m *Message) Encode() []byte
func Decode(m []byte) (*Message, error)

// Names
func NewName(s string) Name
func (n Name) String() string
func (n Name) Equal(o Name) bool
func (n Name) SubdomainOf(other Name) bool

// Addresses
func CreateIPv4(s string) (IPv4, error)
func CreateIPv6(s string) (IPv6, error)
var IPv4Regex, IPv6Regex *regexp.Regexp

// Records: A, AAAA, CNAME, NS, PTR, MX, TXT, SOA, SRV, HINFO, Generic

// Hosts
func ParseHosts(content string) *Hosts
func (h *Hosts) GetAddress(name string) (string, error)
func (h *Hosts) GetName(address string) (string, error)
```

## MRI conformance

Correctness is defined by reference Ruby. A **differential oracle** runs a wide
corpus through both the system `ruby` and this library and compares the results
**byte-for-byte** — not approximated from memory. The oracle tests skip
themselves where `ruby` is not on `PATH` (e.g. the qemu arch lanes), so the
cross-arch builds still validate the library.

## Relationship to Ruby

`go-ruby-resolv/resolv` is **standalone and reusable**, and is the backend bound into
[go-embedded-ruby](https://github.com/go-embedded-ruby/ruby) by `rbgo` as a
native module — the same way [go-ruby-regexp](https://github.com/go-ruby-regexp/regexp) and [go-ruby-marshal](https://github.com/go-ruby-marshal/marshal) are bound. The dependency runs the
other way: this library has no dependency on the Ruby runtime.
