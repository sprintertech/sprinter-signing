// The Licensed Work is (c) 2022 Sygma
// SPDX-License-Identifier: LGPL-3.0-only

package topology

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/rs/zerolog/log"
	"github.com/sprintertech/sprinter-signing/config/relayer"
)

type NetworkTopology struct {
	Peers     []*peer.AddrInfo
	Threshold int
}

func (nt NetworkTopology) IsAllowedPeer(peer peer.ID) bool {
	for _, p := range nt.Peers {
		if p.ID == peer {
			return true
		}
	}

	return false
}

type RawTopology struct {
	Peers     []RawPeer `mapstructure:"Peers" json:"peers"`
	Threshold string    `mapstructure:"Threshold" json:"threshold"`
}

type RawPeer struct {
	PeerAddress string `mapstructure:"PeerAddress" json:"peerAddress"`
}
type S3Client interface {
	GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type Decrypter interface {
	Decrypt(data []byte) []byte
}

type NetworkTopologyProvider interface {
	// NetworkTopology fetches latest topology from network and validates that
	// the version matches expected hash.
	NetworkTopology(hash string) (*NetworkTopology, error)
}

func NewNetworkTopologyProvider(config relayer.TopologyConfiguration, s3Client S3Client, staging bool) (NetworkTopologyProvider, error) {
	decrypter, err := NewAESEncryption([]byte(config.EncryptionKey))
	if err != nil {
		return nil, err
	}

	filename := "production/topology"
	if staging {
		filename = "staging/topology"
	}

	return &TopologyProvider{
		decrypter: decrypter,
		bucket:    config.Url,
		filename:  filename,
		s3Client:  s3Client,
	}, nil
}

type TopologyProvider struct {
	bucket    string
	filename  string
	decrypter Decrypter
	s3Client  S3Client
}

func (t *TopologyProvider) NetworkTopology(hash string) (*NetworkTopology, error) {
	log.Info().Msgf("Reading topology from S3 bucket: %s, file: %s", t.bucket, t.filename)

	output, err := t.s3Client.GetObject(context.Background(), &s3.GetObjectInput{
		Bucket: &t.bucket,
		Key:    &t.filename,
	})
	if err != nil {
		return nil, err
	}

	defer output.Body.Close()
	body, err := io.ReadAll(output.Body)
	if err != nil {
		return nil, err
	}

	response := strings.TrimSuffix(string(body), "\n")
	ct, err := hex.DecodeString(response)
	if err != nil {
		return nil, err
	}
	h := sha256.New()
	h.Write(ct)
	eh := hex.EncodeToString(h.Sum(nil))
	if hash != "" && eh != hash {
		return nil, fmt.Errorf("topology hash %s not matching expected hash %s", string(eh), hash)
	}

	unecryptedBody := t.decrypter.Decrypt(ct)
	rawTopology := &RawTopology{}
	err = json.Unmarshal(unecryptedBody, rawTopology)
	if err != nil {
		return nil, err
	}

	return ProcessRawTopology(rawTopology)
}

func ProcessRawTopology(rawTopology *RawTopology) (*NetworkTopology, error) {
	var peers []*peer.AddrInfo
	for _, p := range rawTopology.Peers {
		addrInfo, err := peer.AddrInfoFromString(p.PeerAddress)
		if err != nil {
			return nil, fmt.Errorf("invalid peer address %s: %w", p.PeerAddress, err)
		}
		peers = append(peers, addrInfo)
	}

	threshold, err := strconv.ParseInt(rawTopology.Threshold, 0, 0)
	if err != nil {
		return nil, fmt.Errorf("unable to parse mpc threshold from topology %v", err)
	}
	if threshold < 1 {
		return nil, fmt.Errorf("mpc threshold must be bigger then 0 %v", err)
	}
	return &NetworkTopology{Peers: peers, Threshold: int(threshold)}, nil
}
