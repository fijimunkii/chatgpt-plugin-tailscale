package tailnet

import (
	"context"
	"errors"
	"net/netip"

	"tailscale.com/client/local"
)

// TSNetSource adapts the embedded node's LocalAPI status into Catalog peers.
type TSNetSource struct {
	client *local.Client
}

func NewTSNetSource(client *local.Client) *TSNetSource {
	return &TSNetSource{client: client}
}

func (s *TSNetSource) Peers(ctx context.Context) ([]Peer, error) {
	if s == nil || s.client == nil {
		return nil, errors.New("Tailscale LocalAPI client is required")
	}
	status, err := s.client.Status(ctx)
	if err != nil {
		return nil, err
	}
	peers := make([]Peer, 0, len(status.Peer))
	for _, peerStatus := range status.Peer {
		if peerStatus == nil {
			continue
		}
		var tags []string
		if peerStatus.Tags != nil {
			tags = peerStatus.Tags.AsSlice()
		}
		keyExpiry := peerStatus.KeyExpiry
		peers = append(peers, Peer{
			ID:          string(peerStatus.ID),
			HostName:    peerStatus.HostName,
			DNSName:     peerStatus.DNSName,
			OS:          peerStatus.OS,
			IPs:         append([]netip.Addr(nil), peerStatus.TailscaleIPs...),
			Tags:        tags,
			Online:      peerStatus.Online,
			Active:      peerStatus.Active,
			Expired:     peerStatus.Expired,
			KeyExpiry:   keyExpiry,
			LastSeen:    peerStatus.LastSeen,
			Shared:      peerStatus.ShareeNode,
			SSHHostKeys: append([]string(nil), peerStatus.SSH_HostKeys...),
		})
	}
	return peers, nil
}
