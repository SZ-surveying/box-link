package netops

import "testing"

func TestParseIfconfigSummary(t *testing.T) {
	output := `en7: flags=8863<UP,BROADCAST,SMART,RUNNING,SIMPLEX,MULTICAST> mtu 1500
	options=400<CHANNEL_IO>
	ether aa:bb:cc:dd:ee:ff
	inet 192.168.2.10 netmask 0xffffff00 broadcast 192.168.2.255
	media: autoselect
	status: active
`

	got := parseIfconfigSummary("en7", output)

	if got.Name != "en7" || got.IPv4 != "192.168.2.10" || got.Netmask != "0xffffff00" || got.Ether != "aa:bb:cc:dd:ee:ff" || got.Status != "active" || got.Media != "autoselect" {
		t.Fatalf("unexpected ifconfig details: %#v", got)
	}
}

func TestParseRouteSummary(t *testing.T) {
	output := `   route to: 192.168.2.20
destination: 192.168.2.20
       mask: 255.255.255.255
    gateway: link#7
  interface: en7
      flags: <UP,HOST,DONE,STATIC>
`

	got := parseRouteSummary("192.168.2.20", output)

	if got.Destination != "192.168.2.20" || got.Gateway != "link#7" || got.Interface != "en7" || got.Flags != "<UP,HOST,DONE,STATIC>" {
		t.Fatalf("unexpected route details: %#v", got)
	}
}

func TestParsePingSummary(t *testing.T) {
	output := `PING 192.168.2.20 (192.168.2.20): 56 data bytes
64 bytes from 192.168.2.20: icmp_seq=0 ttl=64 time=0.431 ms

--- 192.168.2.20 ping statistics ---
1 packets transmitted, 1 packets received, 0.0% packet loss
round-trip min/avg/max/stddev = 0.431/0.431/0.431/0.000 ms
`

	got := parsePingSummary("192.168.2.20", output)

	if got.Target != "192.168.2.20" || got.Responder != "192.168.2.20" || got.Latency != "0.431 ms" || got.PacketLoss != "0.0% packet loss" || got.RoundTrip != "0.431/0.431/0.431/0.000 ms" {
		t.Fatalf("unexpected ping details: %#v", got)
	}
}
