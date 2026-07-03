// Copyright (c) the go-ruby-resolv authors
// SPDX-License-Identifier: BSD-3-Clause
//
// Library-level benchmark driver for the pure-Go go-ruby-resolv library. It
// exercises the OFFLINE, pure-compute primitives of Ruby's resolv — DNS name
// parse/encode, IPv4/IPv6 address parse and render, and DNS message pack/unpack
// over a fixed byte buffer — over an identical, deterministic corpus, so the
// ns/op numbers compare the pure-Go library primitive against each Ruby
// runtime's own stdlib resolv. No sockets and no network are involved: the
// rbgo binding keeps resolv's sockets as stubs, so only the deterministic,
// reproducible compute primitives are measured here.
//
// With CHECK=1 it instead prints one "CHECK\t<label>\t<value>" line per op: an
// integer checksum of the op's result, used to prove the Go output is identical
// to MRI (the oracle) before any timing is trusted.
package main

import (
	"fmt"
	"os"

	resolv "github.com/go-ruby-resolv/resolv"
)

// names is a fixed, realistic set of DNS names, mixing depths, hyphens,
// underscored service labels, an IDN a-label, and mixed case. It is byte-for-
// byte the same list the Ruby workload builds.
var names = []string{
	"www.example.com",
	"mail.example.com",
	"ftp.sub.example.org",
	"a.b.c.d.example.net",
	"host-01.dept.corp.example.com",
	"ns1.iana.org",
	"_sip._tcp.example.com",
	"xn--nxasmq6b.example",
	"long.chain.of.many.labels.deep.in.the.tree.example",
	"EXAMPLE.COM",
	"WWW.Example.Com",
	"localhost",
	"one.two.three.four.five.six.seven.eight",
	"cdn.assets.static.example.io",
	"api-v2.gateway.internal.example.co.uk",
	"a1.b2.c3.d4.e5.f6.example",
}

// ipv4s is a fixed set of dotted-quad addresses spanning the octet range,
// private/loopback/multicast blocks, and the boundary values.
var ipv4s = []string{
	"0.0.0.0", "255.255.255.255", "192.168.1.1", "10.0.0.1",
	"8.8.8.8", "127.0.0.1", "172.16.254.1", "203.0.113.5",
	"1.2.3.4", "93.184.216.34", "100.64.0.1", "224.0.0.251",
	"169.254.0.1", "198.51.100.7", "240.0.0.1", "250.251.252.253",
}

// ipv6s is a fixed set of textual IPv6 addresses covering the forms MRI's
// Resolv::IPv6.create accepts offline: full 8-hex, compressed "::", and the
// embedded-IPv4 mixed forms. (%zone link-local forms are deliberately excluded
// because Resolv::IPv6.create rejects them, exactly as this library does.)
var ipv6s = []string{
	"2001:db8::1",
	"::1",
	"::",
	"2606:2800:220:1:248:1893:25c8:1946",
	"fe80::1",
	"2001:0db8:0000:0000:0000:ff00:0042:8329",
	"::ffff:192.168.1.1",
	"2001:db8:0:0:1:0:0:1",
	"ff02::fb",
	"64:ff9b::1.2.3.4",
	"2001:db8:85a3::8a2e:370:7334",
	"fc00::abcd:1234",
}

// buildMessage constructs one representative DNS response message with records
// spread across the answer, authority and additional sections, exercising every
// RDATA encoder (A, AAAA, CNAME, NS, MX, TXT, SOA, SRV, PTR, HINFO), name
// compression, and the SRV "no compression" path. It is built byte-for-byte the
// same way by the Ruby workload, so the encoded wire is identical.
func buildMessage() *resolv.Message {
	m := resolv.NewMessage(0x1234)
	m.RD = 1
	m.QR = 1
	m.AA = 1
	m.AddQuestion(resolv.NewName("www.example.com"), resolv.TypeA, resolv.ClassIN)

	ip4, _ := resolv.CreateIPv4("93.184.216.34")
	ip6, _ := resolv.CreateIPv6("2606:2800:220:1:248:1893:25c8:1946")
	m.AddAnswer(resolv.NewName("www.example.com"), 3600, resolv.NewCNAME(resolv.NewName("example.com")))
	m.AddAnswer(resolv.NewName("example.com"), 3600, &resolv.A{Address: ip4})
	m.AddAnswer(resolv.NewName("example.com"), 3600, &resolv.AAAA{Address: ip6})
	m.AddAnswer(resolv.NewName("example.com"), 300, &resolv.MX{Preference: 10, Exchange: resolv.NewName("mail.example.com")})
	m.AddAnswer(resolv.NewName("example.com"), 300, &resolv.MX{Preference: 20, Exchange: resolv.NewName("mail2.example.com")})
	m.AddAnswer(resolv.NewName("example.com"), 900, &resolv.TXT{Strings: []string{"v=spf1 include:_spf.example.com ~all"}})

	m.AddAuthority(resolv.NewName("example.com"), 172800, resolv.NewNS(resolv.NewName("ns1.example.com")))
	m.AddAuthority(resolv.NewName("example.com"), 172800, resolv.NewNS(resolv.NewName("ns2.example.com")))
	m.AddAuthority(resolv.NewName("example.com"), 3600, &resolv.SOA{
		MName: resolv.NewName("ns1.example.com"), RName: resolv.NewName("hostmaster.example.com"),
		Serial: 2026070301, Refresh: 7200, Retry: 3600, Expire: 1209600, Minimum: 3600,
	})

	m.AddAdditional(resolv.NewName("ns1.example.com"), 172800, &resolv.A{Address: mustV4("192.0.2.1")})
	m.AddAdditional(resolv.NewName("ns2.example.com"), 172800, &resolv.A{Address: mustV4("192.0.2.2")})
	m.AddAdditional(resolv.NewName("_sip._tcp.example.com"), 300, &resolv.SRV{
		Priority: 10, Weight: 60, Port: 5060, Target: resolv.NewName("sipserver.example.com"),
	})
	m.AddAdditional(resolv.NewName("4.3.2.1.in-addr.arpa"), 3600, resolv.NewPTR(resolv.NewName("host.example.com")))
	m.AddAdditional(resolv.NewName("host.example.com"), 3600, &resolv.HINFO{CPU: "ARM64", OS: "Linux"})
	return m
}

func mustV4(s string) resolv.IPv4 { ip, _ := resolv.CreateIPv4(s); return ip }

// Pre-parsed objects for the to_s (render) ops, so timing measures rendering
// only, not parsing.
var (
	parsedNames = func() []resolv.Name {
		out := make([]resolv.Name, len(names))
		for i, s := range names {
			out[i] = resolv.NewName(s)
		}
		return out
	}()
	parsedV4 = func() []resolv.IPv4 {
		out := make([]resolv.IPv4, len(ipv4s))
		for i, s := range ipv4s {
			out[i], _ = resolv.CreateIPv4(s)
		}
		return out
	}()
	parsedV6 = func() []resolv.IPv6 {
		out := make([]resolv.IPv6, len(ipv6s))
		for i, s := range ipv6s {
			out[i], _ = resolv.CreateIPv6(s)
		}
		return out
	}()
	fixedMessage = buildMessage()
	fixedWire    = fixedMessage.Encode()
)

// opNameParse parses every name; checksum = sum of label counts.
func opNameParse() int {
	acc := 0
	for _, s := range names {
		acc += resolv.NewName(s).Length()
	}
	return acc
}

// opNameToS renders every pre-parsed name; checksum = total rendered bytes.
func opNameToS() int {
	acc := 0
	for _, n := range parsedNames {
		acc += len(n.String())
	}
	return acc
}

// opV4Parse parses every dotted-quad; checksum = sum of all address bytes.
func opV4Parse() int {
	acc := 0
	for _, s := range ipv4s {
		ip, _ := resolv.CreateIPv4(s)
		for _, b := range ip.Addr {
			acc += int(b)
		}
	}
	return acc
}

// opV4ToS renders every pre-parsed IPv4; checksum = total rendered bytes.
func opV4ToS() int {
	acc := 0
	for _, ip := range parsedV4 {
		acc += len(ip.String())
	}
	return acc
}

// opV6Parse parses every textual IPv6; checksum = sum of all address bytes.
func opV6Parse() int {
	acc := 0
	for _, s := range ipv6s {
		ip, _ := resolv.CreateIPv6(s)
		for _, b := range ip.Addr {
			acc += int(b)
		}
	}
	return acc
}

// opV6ToS renders every pre-parsed IPv6 (exercising zero-run compression);
// checksum = total rendered bytes.
func opV6ToS() int {
	acc := 0
	for _, ip := range parsedV6 {
		acc += len(ip.String())
	}
	return acc
}

// opMsgEncode encodes the fixed message to wire; checksum = sum of wire bytes
// (a strong cross-implementation equality check: it only matches MRI if every
// field, RDATA encoder and name-compression pointer agrees).
func opMsgEncode() int {
	w := fixedMessage.Encode()
	acc := 0
	for _, b := range w {
		acc += int(b)
	}
	return acc
}

// opMsgDecode decodes the fixed wire buffer; checksum = question count plus,
// for every resource record, ttl + record TYPE (proves the sections and typed
// RDATA decoded structurally).
func opMsgDecode() int {
	msg, err := resolv.Decode(fixedWire)
	if err != nil {
		panic(err)
	}
	acc := len(msg.Question)
	for _, sec := range [][]resolv.RR{msg.Answer, msg.Authority, msg.Additional} {
		for _, rr := range sec {
			acc += int(rr.TTL) + int(rr.Data.TypeValue())
		}
	}
	return acc
}

var ops = []struct {
	label string
	fn    func() int
}{
	{"name-parse", opNameParse},
	{"name-to_s", opNameToS},
	{"ipv4-parse", opV4Parse},
	{"ipv4-to_s", opV4ToS},
	{"ipv6-parse", opV6Parse},
	{"ipv6-to_s", opV6ToS},
	{"message-encode", opMsgEncode},
	{"message-decode", opMsgDecode},
}

func main() {
	if os.Getenv("CHECK") != "" {
		for _, o := range ops {
			fmt.Printf("CHECK\t%s\t%d\n", o.label, o.fn())
		}
		return
	}
	const inner = 100
	for _, o := range ops {
		fn := o.fn
		bench(o.label, inner, func() { sink = fn() })
	}
}
