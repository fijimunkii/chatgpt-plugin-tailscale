package tailnet

import (
	"context"
	"net/netip"
	"strings"
	"testing"
)

type fakeSource []Peer

func (f fakeSource) Peers(context.Context) ([]Peer, error) {
	return append([]Peer(nil), f...), nil
}

func readyPeer() Peer {
	return Peer{
		ID:          "n123",
		HostName:    "app-1",
		DNSName:     "app-1.example.ts.net.",
		OS:          "linux",
		IPs:         []netip.Addr{netip.MustParseAddr("100.64.0.10")},
		Tags:        []string{"tag:chatgpt-managed"},
		Online:      true,
		SSHHostKeys: []string{"ssh-ed25519 AAAATEST"},
	}
}

func TestCatalogFiltersByTag(t *testing.T) {
	allowed := readyPeer()
	denied := readyPeer()
	denied.ID = "n456"
	denied.HostName = "db-1"
	denied.DNSName = "db-1.example.ts.net."
	denied.Tags = []string{"tag:database"}

	catalog, err := NewCatalog(fakeSource{denied, allowed}, Policy{AllowedTags: []string{"tag:chatgpt-managed"}})
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	hosts, err := catalog.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(hosts) != 1 || hosts[0].ID != allowed.ID {
		t.Fatalf("unexpected hosts: %+v", hosts)
	}
}

func TestCatalogExplicitHostOverridesTag(t *testing.T) {
	peer := readyPeer()
	peer.Tags = nil
	catalog, err := NewCatalog(fakeSource{peer}, Policy{AllowedHosts: []string{"app-1.example.ts.net"}})
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	hosts, err := catalog.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(hosts) != 1 {
		t.Fatalf("expected one host, got %d", len(hosts))
	}
}

func TestResolveAcceptsShortMagicDNSAndPrefersIPv4(t *testing.T) {
	peer := readyPeer()
	peer.IPs = []netip.Addr{
		netip.MustParseAddr("fd7a:115c:a1e0::10"),
		netip.MustParseAddr("100.64.0.10"),
	}
	catalog, _ := NewCatalog(fakeSource{peer}, Policy{AllowedTags: []string{"tag:chatgpt-managed"}})
	target, err := catalog.Resolve(context.Background(), "APP-1")
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if got := target.DialIP.String(); got != "100.64.0.10" {
		t.Fatalf("dial IP = %q", got)
	}
	if !target.ReadyForInspection {
		t.Fatalf("target should be ready: %+v", target.Host)
	}
}

func TestResolveRejectsAmbiguousShortName(t *testing.T) {
	one := readyPeer()
	two := readyPeer()
	two.ID = "n456"
	two.HostName = "app-1"
	two.DNSName = "app-1.other.ts.net."
	two.IPs = []netip.Addr{netip.MustParseAddr("100.64.0.11")}
	catalog, _ := NewCatalog(fakeSource{one, two}, Policy{AllowedTags: []string{"tag:chatgpt-managed"}})
	_, err := catalog.Resolve(context.Background(), "app-1")
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguous error, got %v", err)
	}
}

func TestUnavailableReasons(t *testing.T) {
	peer := readyPeer()
	peer.Online = false
	catalog, _ := NewCatalog(fakeSource{peer}, Policy{AllowedTags: []string{"tag:chatgpt-managed"}})
	hosts, err := catalog.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if hosts[0].ReadyForInspection || hosts[0].UnavailableReason != "offline" {
		t.Fatalf("unexpected host availability: %+v", hosts[0])
	}
}

func TestSharedNodesExcludedByDefault(t *testing.T) {
	peer := readyPeer()
	peer.Shared = true
	catalog, _ := NewCatalog(fakeSource{peer}, Policy{AllowedTags: []string{"tag:chatgpt-managed"}})
	hosts, err := catalog.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(hosts) != 0 {
		t.Fatalf("shared host leaked through policy: %+v", hosts)
	}
}

func TestCatalogRejectsAmbiguousConfiguredHostSelector(t *testing.T) {
	one := readyPeer()
	two := readyPeer()
	two.ID = "n456"
	two.DNSName = "app-1.other.ts.net."
	two.IPs = []netip.Addr{netip.MustParseAddr("100.64.0.11")}
	one.Tags = nil
	two.Tags = nil

	catalog, err := NewCatalog(fakeSource{one, two}, Policy{AllowedHosts: []string{"app-1"}})
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	_, err = catalog.List(context.Background())
	if err == nil || !strings.Contains(err.Error(), "configured host selector") || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguous selector error, got %v", err)
	}
}

func TestCatalogCopiesAndNormalizesPolicy(t *testing.T) {
	peer := readyPeer()
	tags := []string{" tag:chatgpt-managed ", "tag:chatgpt-managed"}
	catalog, err := NewCatalog(fakeSource{peer}, Policy{AllowedTags: tags})
	if err != nil {
		t.Fatalf("NewCatalog: %v", err)
	}
	tags[0] = "tag:mutated"
	hosts, err := catalog.List(context.Background())
	if err != nil || len(hosts) != 1 {
		t.Fatalf("catalog policy mutated or failed: hosts=%+v err=%v", hosts, err)
	}
}
