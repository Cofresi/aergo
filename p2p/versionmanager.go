/*
 * @file
 * @copyright defined in aergo/LICENSE.txt
 */

package p2p

import (
	"fmt"
	"github.com/aergoio/aergo-lib/log"
	"github.com/aergoio/aergo/chain"
	"github.com/aergoio/aergo/p2p/p2pcommon"
	v030 "github.com/aergoio/aergo/p2p/v030"
	"github.com/aergoio/aergo/types"
	"io"
)

type defaultVersionManager struct {
	pm     p2pcommon.PeerManager
	actor  p2pcommon.ActorService
	ca     types.ChainAccessor
	logger *log.Logger

	// check if is it ad hoc
	localChainID *types.ChainID
}

func newDefaultVersionManager(pm p2pcommon.PeerManager, actor p2pcommon.ActorService, ca types.ChainAccessor, logger *log.Logger, localChainID *types.ChainID) *defaultVersionManager {
	return &defaultVersionManager{pm: pm, actor: actor, ca:ca, logger: logger, localChainID: localChainID}
}

func (vm *defaultVersionManager) FindBestP2PVersion(versions []p2pcommon.P2PVersion) p2pcommon.P2PVersion {
	for _, supported := range AcceptedInboundVersions {
		for _, reqVer := range versions {
			if supported == reqVer {
				return reqVer
			}
		}
	}
	return p2pcommon.P2PVersionUnknown
}

func (h *defaultVersionManager) GetVersionedHandshaker(version p2pcommon.P2PVersion, peerID types.PeerID, rwc io.ReadWriteCloser) (p2pcommon.VersionedHandshaker, error) {
	switch version {
	case p2pcommon.P2PVersion032:
		vhs := v030.NewV032VersionedHS(h.pm, h.actor, h.logger, h.localChainID, peerID, rwc, chain.Genesis.Block().Hash)
		return vhs, nil
	case p2pcommon.P2PVersion031:
		v030hs := v030.NewV030VersionedHS(h.pm, h.actor, h.logger, h.localChainID, peerID, rwc)
		return v030hs, nil
	case p2pcommon.P2PVersion030:
		v030hs := v030.NewV030VersionedHS(h.pm, h.actor, h.logger, h.localChainID, peerID, rwc)
		return v030hs, nil
	default:
		return nil, fmt.Errorf("not supported version")
	}
}