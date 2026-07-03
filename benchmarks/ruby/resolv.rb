# frozen_string_literal: true
# Copyright (c) the go-ruby-resolv authors
# SPDX-License-Identifier: BSD-3-Clause
#
# Reference resolv workload, mirroring benchmarks/go/main.go op-for-op over an
# identical, deterministic corpus. It exercises only the OFFLINE, pure-compute
# primitives of Ruby's resolv — DNS name parse/encode, IPv4/IPv6 address parse
# and render, and DNS message pack/unpack over a fixed byte buffer. No sockets
# and no network are touched, so every op is deterministic and reproducible.
#
# Run normally it reports ns/op per op through the shared harness; run with
# CHECK=1 it prints one "CHECK\t<label>\t<value>" line per op so the Go output
# can be proven identical to MRI (the oracle) before any timing is trusted.
require "resolv"
require_relative "_harness"

# Byte-for-byte the same lists the Go driver builds.
NAMES = %w[
  www.example.com
  mail.example.com
  ftp.sub.example.org
  a.b.c.d.example.net
  host-01.dept.corp.example.com
  ns1.iana.org
  _sip._tcp.example.com
  xn--nxasmq6b.example
  long.chain.of.many.labels.deep.in.the.tree.example
  EXAMPLE.COM
  WWW.Example.Com
  localhost
  one.two.three.four.five.six.seven.eight
  cdn.assets.static.example.io
  api-v2.gateway.internal.example.co.uk
  a1.b2.c3.d4.e5.f6.example
].freeze

IPV4S = %w[
  0.0.0.0 255.255.255.255 192.168.1.1 10.0.0.1
  8.8.8.8 127.0.0.1 172.16.254.1 203.0.113.5
  1.2.3.4 93.184.216.34 100.64.0.1 224.0.0.251
  169.254.0.1 198.51.100.7 240.0.0.1 250.251.252.253
].freeze

IPV6S = %w[
  2001:db8::1
  ::1
  ::
  2606:2800:220:1:248:1893:25c8:1946
  fe80::1
  2001:0db8:0000:0000:0000:ff00:0042:8329
  ::ffff:192.168.1.1
  2001:db8:0:0:1:0:0:1
  ff02::fb
  64:ff9b::1.2.3.4
  2001:db8:85a3::8a2e:370:7334
  fc00::abcd:1234
].freeze

# build_message constructs the same representative DNS response message the Go
# driver builds, records spread across the answer/authority/additional sections.
def build_message
  n = ->(s) { Resolv::DNS::Name.create(s) }
  m = Resolv::DNS::Message.new(0x1234)
  m.rd = 1
  m.qr = 1
  m.aa = 1
  m.add_question(n.call("www.example.com"), Resolv::DNS::Resource::IN::A)

  m.add_answer(n.call("www.example.com"), 3600, Resolv::DNS::Resource::IN::CNAME.new(n.call("example.com")))
  m.add_answer(n.call("example.com"), 3600, Resolv::DNS::Resource::IN::A.new(Resolv::IPv4.create("93.184.216.34")))
  m.add_answer(n.call("example.com"), 3600, Resolv::DNS::Resource::IN::AAAA.new(Resolv::IPv6.create("2606:2800:220:1:248:1893:25c8:1946")))
  m.add_answer(n.call("example.com"), 300, Resolv::DNS::Resource::IN::MX.new(10, n.call("mail.example.com")))
  m.add_answer(n.call("example.com"), 300, Resolv::DNS::Resource::IN::MX.new(20, n.call("mail2.example.com")))
  m.add_answer(n.call("example.com"), 900, Resolv::DNS::Resource::IN::TXT.new("v=spf1 include:_spf.example.com ~all"))

  m.add_authority(n.call("example.com"), 172800, Resolv::DNS::Resource::IN::NS.new(n.call("ns1.example.com")))
  m.add_authority(n.call("example.com"), 172800, Resolv::DNS::Resource::IN::NS.new(n.call("ns2.example.com")))
  m.add_authority(n.call("example.com"), 3600, Resolv::DNS::Resource::IN::SOA.new(
    n.call("ns1.example.com"), n.call("hostmaster.example.com"),
    2026070301, 7200, 3600, 1209600, 3600
  ))

  m.add_additional(n.call("ns1.example.com"), 172800, Resolv::DNS::Resource::IN::A.new(Resolv::IPv4.create("192.0.2.1")))
  m.add_additional(n.call("ns2.example.com"), 172800, Resolv::DNS::Resource::IN::A.new(Resolv::IPv4.create("192.0.2.2")))
  m.add_additional(n.call("_sip._tcp.example.com"), 300, Resolv::DNS::Resource::IN::SRV.new(10, 60, 5060, n.call("sipserver.example.com")))
  m.add_additional(n.call("4.3.2.1.in-addr.arpa"), 3600, Resolv::DNS::Resource::IN::PTR.new(n.call("host.example.com")))
  m.add_additional(n.call("host.example.com"), 3600, Resolv::DNS::Resource::IN::HINFO.new("ARM64", "Linux"))
  m
end

# Pre-parsed objects for the render (to_s) ops, so timing measures rendering
# only, not parsing. The fixed message is encoded once to the wire buffer that
# the decode op consumes.
PARSED_NAMES = NAMES.map { |s| Resolv::DNS::Name.create(s) }.freeze
PARSED_V4    = IPV4S.map { |s| Resolv::IPv4.create(s) }.freeze
PARSED_V6    = IPV6S.map { |s| Resolv::IPv6.create(s) }.freeze
FIXED_MESSAGE = build_message
FIXED_WIRE    = FIXED_MESSAGE.encode.freeze

# name-parse: parse every name; checksum = sum of label counts.
def op_name_parse
  acc = 0
  NAMES.each { |s| acc += Resolv::DNS::Name.create(s).length }
  acc
end

# name-to_s: render every pre-parsed name; checksum = total rendered bytes.
def op_name_to_s
  acc = 0
  PARSED_NAMES.each { |n| acc += n.to_s.bytesize }
  acc
end

# ipv4-parse: parse every dotted-quad; checksum = sum of all address bytes.
def op_v4_parse
  acc = 0
  IPV4S.each { |s| Resolv::IPv4.create(s).address.each_byte { |b| acc += b } }
  acc
end

# ipv4-to_s: render every pre-parsed IPv4; checksum = total rendered bytes.
def op_v4_to_s
  acc = 0
  PARSED_V4.each { |ip| acc += ip.to_s.bytesize }
  acc
end

# ipv6-parse: parse every textual IPv6; checksum = sum of all address bytes.
def op_v6_parse
  acc = 0
  IPV6S.each { |s| Resolv::IPv6.create(s).address.each_byte { |b| acc += b } }
  acc
end

# ipv6-to_s: render every pre-parsed IPv6 (zero-run compression); checksum =
# total rendered bytes.
def op_v6_to_s
  acc = 0
  PARSED_V6.each { |ip| acc += ip.to_s.bytesize }
  acc
end

# message-encode: encode the fixed message to wire; checksum = sum of wire bytes.
def op_msg_encode
  acc = 0
  FIXED_MESSAGE.encode.each_byte { |b| acc += b }
  acc
end

# message-decode: decode the fixed wire buffer; checksum = question count plus,
# for every resource record, ttl + record TYPE.
def op_msg_decode
  msg = Resolv::DNS::Message.decode(FIXED_WIRE)
  acc = 0
  msg.each_question { |_name, _tc| acc += 1 }
  [:each_answer, :each_authority, :each_additional].each do |m|
    msg.send(m) { |_name, ttl, data| acc += ttl + data.class::TypeValue }
  end
  acc
end

OPS = [
  ["name-parse",     method(:op_name_parse)],
  ["name-to_s",      method(:op_name_to_s)],
  ["ipv4-parse",     method(:op_v4_parse)],
  ["ipv4-to_s",      method(:op_v4_to_s)],
  ["ipv6-parse",     method(:op_v6_parse)],
  ["ipv6-to_s",      method(:op_v6_to_s)],
  ["message-encode", method(:op_msg_encode)],
  ["message-decode", method(:op_msg_decode)],
].freeze

if ENV["CHECK"] && !ENV["CHECK"].empty?
  OPS.each { |label, m| printf("CHECK\t%s\t%d\n", label, m.call) }
else
  INNER = 100
  OPS.each { |label, m| bench(label, INNER) { m.call } }
end
