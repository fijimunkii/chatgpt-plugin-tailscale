package tailnet

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"sort"
	"strings"
	"time"
)

// Peer is the subset of Tailscale peer status needed by the gateway.
type Peer struct {
	ID          string
	HostName    string
	DNSName     string
	OS          string
	IPs         []netip.Addr
	Tags        []string
	Online      bool
	Active      bool
	Expired     bool
	KeyExpiry   *time.Time
	LastSeen    time.Time
	Shared      bool
	SSHHostKeys []string
}

// Source supplies a fresh view of the gateway's tailnet peers.
type Source interface {
	Peers(context.Context) ([]Peer, error)
}

// Policy restricts which peers can ever be returned or targeted.
type Policy struct {
	AllowedTags      []string
	AllowedHosts     []string
	AllowSharedNodes bool
}

// Host is the safe, serializable peer representation returned by MCP tools.
type Host struct {
	ID                 string     `json:"id"`
	HostName           string     `json:"hostname,omitempty"`
	DNSName            string     `json:"dns_name,omitempty"`
	OS                 string     `json:"os,omitempty"`
	IPs                []string   `json:"ips"`
	Tags               []string   `json:"tags"`
	Online             bool       `json:"online"`
	Active             bool       `json:"active"`
	Expired            bool       `json:"expired"`
	KeyExpiry          *time.Time `json:"key_expiry,omitempty"`
	LastSeen           *time.Time `json:"last_seen,omitempty"`
	Shared             bool       `json:"shared"`
	SSHHostKeyCount    int        `json:"ssh_host_key_count"`
	ReadyForInspection bool       `json:"ready_for_inspection"`
	UnavailableReason  string     `json:"unavailable_reason,omitempty"`
}

// Target contains the verified routing material needed by the SSH runner.
type Target struct {
	Host
	DialIP      netip.Addr `json:"-"`
	SSHHostKeys []string   `json:"-"`
}

// Catalog applies the local allowlist before exposing peers.
type Catalog struct {
	source Source
	policy Policy
}

func NewCatalog(source Source, policy Policy) (*Catalog, error) {
	if source == nil {
		return nil, errors.New("tailnet source is required")
	}
	policy.AllowedTags = uniqueValues(policy.AllowedTags, strings.TrimSpace)
	policy.AllowedHosts = uniqueValues(policy.AllowedHosts, normalizeHost)
	if len(policy.AllowedTags) == 0 && len(policy.AllowedHosts) == 0 {
		return nil, errors.New("tailnet policy requires at least one allowed tag or host")
	}
	return &Catalog{source: source, policy: policy}, nil
}

// List returns only peers permitted by Policy.
func (c *Catalog) List(ctx context.Context) ([]Host, error) {
	peers, err := c.allowedPeers(ctx)
	if err != nil {
		return nil, err
	}
	hosts := make([]Host, 0, len(peers))
	for _, peer := range peers {
		hosts = append(hosts, hostFromPeer(peer))
	}
	sort.Slice(hosts, func(i, j int) bool {
		return displayName(hosts[i]) < displayName(hosts[j])
	})
	return hosts, nil
}

// Resolve performs an exact, case-insensitive lookup across stable ID,
// hostname, MagicDNS name, short MagicDNS name, and Tailscale IP.
func (c *Catalog) Resolve(ctx context.Context, query string) (Target, error) {
	query = normalizeHost(query)
	if query == "" {
		return Target{}, errors.New("host is required")
	}
	peers, err := c.allowedPeers(ctx)
	if err != nil {
		return Target{}, err
	}

	matches := make([]Peer, 0, 1)
	for _, peer := range peers {
		if peerMatches(peer, query) {
			matches = append(matches, peer)
		}
	}
	if len(matches) == 0 {
		return Target{}, fmt.Errorf("host %q is not present in the configured tailnet allowlist", query)
	}
	if len(matches) > 1 {
		names := make([]string, 0, len(matches))
		for _, peer := range matches {
			names = append(names, preferredPeerName(peer))
		}
		sort.Strings(names)
		return Target{}, fmt.Errorf("host %q is ambiguous; use a full MagicDNS name, Tailscale IP, or stable ID (matches: %s)", query, strings.Join(names, ", "))
	}

	peer := matches[0]
	host := hostFromPeer(peer)
	ip := preferredIP(peer.IPs)
	return Target{
		Host:        host,
		DialIP:      ip,
		SSHHostKeys: append([]string(nil), peer.SSHHostKeys...),
	}, nil
}

func (c *Catalog) allowedPeers(ctx context.Context) ([]Peer, error) {
	peers, err := c.source.Peers(ctx)
	if err != nil {
		return nil, fmt.Errorf("read tailnet status: %w", err)
	}

	candidates := make([]Peer, 0, len(peers))
	for _, peer := range peers {
		if peer.Shared && !c.policy.AllowSharedNodes {
			continue
		}
		candidates = append(candidates, peer)
	}

	// Resolve explicit selectors against the whole candidate set before
	// admitting peers. A short hostname that matches multiple nodes must fail
	// closed rather than silently widening the allowlist.
	explicit := make(map[int]struct{})
	for _, selector := range c.policy.AllowedHosts {
		matches := make([]int, 0, 1)
		for i, peer := range candidates {
			if peerMatches(peer, selector) {
				matches = append(matches, i)
			}
		}
		if len(matches) > 1 {
			names := make([]string, 0, len(matches))
			for _, i := range matches {
				names = append(names, preferredPeerName(candidates[i]))
			}
			sort.Strings(names)
			return nil, fmt.Errorf("configured host selector %q is ambiguous; use a full MagicDNS name, Tailscale IP, or stable ID (matches: %s)", selector, strings.Join(names, ", "))
		}
		if len(matches) == 1 {
			explicit[matches[0]] = struct{}{}
		}
	}

	out := make([]Peer, 0, len(candidates))
	for i, peer := range candidates {
		_, explicitlyAllowed := explicit[i]
		if explicitlyAllowed || c.policy.allowsTag(peer) {
			out = append(out, peer)
		}
	}
	return out, nil
}

func (p Policy) allowsTag(peer Peer) bool {
	for _, peerTag := range peer.Tags {
		for _, allowedTag := range p.AllowedTags {
			if peerTag == allowedTag {
				return true
			}
		}
	}
	return false
}

func peerMatches(peer Peer, normalizedQuery string) bool {
	if normalizedQuery == "" {
		return false
	}
	for _, alias := range peerAliases(peer) {
		if normalizeHost(alias) == normalizedQuery {
			return true
		}
	}
	return false
}

func peerAliases(peer Peer) []string {
	aliases := []string{peer.ID, peer.HostName, strings.TrimSuffix(peer.DNSName, ".")}
	dns := strings.TrimSuffix(peer.DNSName, ".")
	if i := strings.IndexByte(dns, '.'); i > 0 {
		aliases = append(aliases, dns[:i])
	}
	for _, ip := range peer.IPs {
		if ip.IsValid() {
			aliases = append(aliases, ip.String())
		}
	}
	return aliases
}

func hostFromPeer(peer Peer) Host {
	ips := make([]string, 0, len(peer.IPs))
	for _, ip := range peer.IPs {
		if ip.IsValid() {
			ips = append(ips, ip.String())
		}
	}
	sort.Strings(ips)
	tags := append([]string(nil), peer.Tags...)
	sort.Strings(tags)

	reason := unavailableReason(peer)
	var lastSeen *time.Time
	if !peer.LastSeen.IsZero() {
		t := peer.LastSeen.UTC()
		lastSeen = &t
	}
	var keyExpiry *time.Time
	if peer.KeyExpiry != nil {
		t := peer.KeyExpiry.UTC()
		keyExpiry = &t
	}
	return Host{
		ID:                 peer.ID,
		HostName:           peer.HostName,
		DNSName:            strings.TrimSuffix(peer.DNSName, "."),
		OS:                 peer.OS,
		IPs:                ips,
		Tags:               tags,
		Online:             peer.Online,
		Active:             peer.Active,
		Expired:            peer.Expired,
		KeyExpiry:          keyExpiry,
		LastSeen:           lastSeen,
		Shared:             peer.Shared,
		SSHHostKeyCount:    len(peer.SSHHostKeys),
		ReadyForInspection: reason == "",
		UnavailableReason:  reason,
	}
}

func unavailableReason(peer Peer) string {
	if !strings.EqualFold(peer.OS, "linux") {
		return "unsupported_os"
	}
	if peer.Expired {
		return "node_key_expired"
	}
	if !peer.Online {
		return "offline"
	}
	if !preferredIP(peer.IPs).IsValid() {
		return "no_tailscale_ip"
	}
	if len(peer.SSHHostKeys) == 0 {
		return "tailscale_ssh_host_keys_unavailable"
	}
	return ""
}

func preferredIP(ips []netip.Addr) netip.Addr {
	for _, ip := range ips {
		if ip.IsValid() && ip.Is4() {
			return ip
		}
	}
	for _, ip := range ips {
		if ip.IsValid() {
			return ip
		}
	}
	return netip.Addr{}
}

func preferredPeerName(peer Peer) string {
	if dns := strings.TrimSuffix(peer.DNSName, "."); dns != "" {
		return dns
	}
	if peer.HostName != "" {
		return peer.HostName
	}
	return peer.ID
}

func displayName(host Host) string {
	if host.DNSName != "" {
		return strings.ToLower(host.DNSName)
	}
	if host.HostName != "" {
		return strings.ToLower(host.HostName)
	}
	return strings.ToLower(host.ID)
}

func uniqueValues(values []string, normalize func(string) string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = normalize(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func normalizeHost(v string) string {
	return strings.ToLower(strings.TrimSuffix(strings.TrimSpace(v), "."))
}
