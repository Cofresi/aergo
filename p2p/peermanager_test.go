package p2p

import (
	"fmt"
	"github.com/libp2p/go-libp2p-core/network"
	"github.com/pkg/errors"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Cofresi/aergo-lib/log"
	cfg "github.com/aergoio/aergo/config"
	"github.com/aergoio/aergo/message"
	"github.com/aergoio/aergo/p2p/p2pcommon"
	"github.com/aergoio/aergo/p2p/p2pkey"
	"github.com/aergoio/aergo/p2p/p2pmock"
	"github.com/aergoio/aergo/p2p/p2putil"
	"github.com/aergoio/aergo/types"
	"github.com/golang/mock/gomock"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/stretchr/testify/assert"
)

func FailTestGetPeers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockActor := p2pmock.NewMockActorService(ctrl)
	dummyBlock := types.Block{Hash: dummyBlockHash, Header: &types.BlockHeader{BlockNo: dummyBlockHeight}}
	mockActor.EXPECT().CallRequest(gomock.Any(), gomock.Any(), gomock.Any()).
		Return(message.GetBlockRsp{Block: &dummyBlock}, nil)
	target := NewPeerManager(nil, nil, mockActor, nil, nil, nil, nil, log.NewLogger("test.p2p"), cfg.NewServerContext("", "").GetDefaultConfig().(*cfg.Config), false).(*peerManager)

	iterSize := 500
	wg := sync.WaitGroup{}
	waitChan := make(chan int)
	wg.Add(1)
	go func() {
		for i := 0; i < iterSize; i++ {
			peerID := types.PeerID(strconv.Itoa(i))
			peerMeta := p2pcommon.PeerMeta{ID: peerID}
			target.remotePeers[peerID] = newRemotePeer(peerMeta, 0, target, mockActor, logger, nil, nil, nil)
			if i == (iterSize >> 2) {
				wg.Done()
			}
		}
	}()

	go func() {
		wg.Wait()
		for key, val := range target.remotePeers {
			fmt.Printf("%s is %s\n", key.String(), val.State().String())
		}
		waitChan <- 0
	}()

	<-waitChan
}

func TestPeerManager_GetPeers(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockActorServ := p2pmock.NewMockActorService(ctrl)

	tLogger := log.NewLogger("test.p2p")
	tConfig := cfg.NewServerContext("", "").GetDefaultConfig().(*cfg.Config)
	p2pkey.InitNodeInfo(&tConfig.BaseConfig, tConfig.P2P, "1.0.0-test", tLogger)
	target := NewPeerManager(nil, nil, mockActorServ, nil, nil, nil, nil, tLogger, tConfig, false).(*peerManager)

	iterSize := 500
	wg := &sync.WaitGroup{}
	wgAll := &sync.WaitGroup{}
	waitChan := make(chan int)
	wg.Add(1)
	wgAll.Add(1)
	go func() {
		for i := 0; i < iterSize; i++ {
			peerID := types.PeerID(strconv.Itoa(i))
			peerMeta := p2pcommon.PeerMeta{ID: peerID}
			target.insertPeer(peerID, newRemotePeer(peerMeta, 0, target, mockActorServ, logger, nil, nil, nil))
			if i == (iterSize >> 2) {
				wg.Done()
			}
		}
		wgAll.Done()
	}()

	cnt := 0
	go func() {
		wg.Wait()
		for _ = range target.GetPeers() {
			cnt++
		}
		assert.True(t, cnt > (iterSize >> 2))
		waitChan <- 0
	}()

	<-waitChan

	wgAll.Wait()
	assert.True(t, iterSize == len(target.GetPeers()))
}

func TestPeerManager_GetPeerAddresses(t *testing.T) {
	peersLen := 6
	hiddenCnt := 3
	samplePeers := make([]*remotePeerImpl, peersLen)
	for i := 0; i < peersLen; i++ {
		pkey, _, _ := crypto.GenerateKeyPair(crypto.Secp256k1, 256)
		pid, _ := types.IDFromPrivateKey(pkey)
		samplePeers[i] = &remotePeerImpl{meta: p2pcommon.PeerMeta{ID: pid, Hidden: i < hiddenCnt}, lastStatus: &types.LastBlockStatus{}}
	}

	tests := []struct {
		name string

		hidden   bool
		showSelf bool

		wantCnt int
	}{
		{"TDefault", false, false, peersLen},
		{"TWSelf", false, true, peersLen + 1},
		{"TWOHidden", true, false, peersLen - hiddenCnt},
		{"TWOHiddenWSelf", false, true, peersLen - hiddenCnt + 1},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			pm := &peerManager{
				remotePeers: make(map[types.PeerID]p2pcommon.RemotePeer),
				mutex:       &sync.Mutex{},
			}
			for _, peer := range samplePeers {
				pm.remotePeers[peer.ID()] = peer
			}
			pm.updatePeerCache()

			actPeers := pm.GetPeerAddresses(false, false)
			assert.Equal(t, peersLen, len(actPeers))
		})
	}
}

func TestPeerManager_init(t *testing.T) {
	tConfig := cfg.NewServerContext("", "").GetDefaultConfig().(*cfg.Config)
	defaultCfg := tConfig.P2P
	p2pkey.InitNodeInfo(&tConfig.BaseConfig, defaultCfg, "1.0.0-test", logger)
	localIP, _ := p2putil.ExternalIP()

	tests := []struct {
		name            string
		inCfg           *cfg.P2PConfig
		expectProtoAddr string
		expectProtoPort uint32
		expectBindAddr  string
		expectBindPort  uint32
		expectPanic     bool
	}{
		{"TDefault", defaultCfg, localIP.String(), uint32(defaultCfg.NetProtocolPort), localIP.String(), uint32(defaultCfg.NetProtocolPort), false},
		// wrong ProtocolAddress 0.0.0.0
		{"TUnspecifiedAddr", &cfg.P2PConfig{NetProtocolAddr: "0.0.0.0", NetProtocolPort: 7846}, localIP.String(), 7846, localIP.String(), uint32(defaultCfg.NetProtocolPort), true},
		// wrong ProtocolAddress
		{"TWrongAddr", &cfg.P2PConfig{NetProtocolAddr: "24558.30.0.0", NetProtocolPort: 7846}, localIP.String(), 7846, localIP.String(), 7846, true},
		// bind all address
		{"TBindAll", &cfg.P2PConfig{NetProtocolAddr: "", NetProtocolPort: 7846, NPBindAddr: "0.0.0.0"}, localIP.String(), 7846, "0.0.0.0", 7846, false},
		// bind different address
		{"TBindDifferAddr", &cfg.P2PConfig{NetProtocolAddr: "", NetProtocolPort: 7846, NPBindAddr: "172.21.1.2"}, localIP.String(), 7846, "172.21.1.2", 7846, false},
		// bind different port
		{"TDifferPort", &cfg.P2PConfig{NetProtocolAddr: "", NetProtocolPort: 7846, NPBindPort: 12345}, localIP.String(), 7846, localIP.String(), 12345, false},
		// bind different address and port
		{"TBindDiffer", &cfg.P2PConfig{NetProtocolAddr: "", NetProtocolPort: 7846, NPBindAddr: "172.21.1.2", NPBindPort: 12345}, localIP.String(), 7846, "172.21.1.2", 12345, false},
		// TODO: test cases
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.expectPanic {
				defer func() {
					if r := recover(); r != nil {
						fmt.Println(test.name, " expected panic occurred ", r)
					}
				}()
				pm := peerManager{conf: test.inCfg}

				pm.init()
			}
		})
	}
}

func Test_peerManager_runManagePeers_MultiConnWorks(t *testing.T) {
	// Test if it works well when concurrent connections is handshaked.
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	logger := log.NewLogger("p2p.test")
	type desc struct {
		pid      types.PeerID
		outbound bool
		hsTime   time.Duration
	}
	ds := make([]desc, 10)
	for i := 0; i < 10; i++ {
		pkey, _, _ := crypto.GenerateKeyPair(crypto.Secp256k1, 256)
		pid, _ := types.IDFromPrivateKey(pkey)
		ds[i] = desc{hsTime: time.Millisecond * 10, outbound: true, pid: pid}
	}
	tests := []struct {
		name string

		conns []desc
	}{
		{"T10", ds},
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockPeerFinder := p2pmock.NewMockPeerFinder(ctrl)
			mockWPManager := p2pmock.NewMockWaitingPeerManager(ctrl)
			mockWPManager.EXPECT().CheckAndConnect().AnyTimes()
			mockNT := p2pmock.NewMockNetworkTransport(ctrl)
			mockNT.EXPECT().AddStreamHandler(gomock.Any(), gomock.Any()).AnyTimes()
			mockNT.EXPECT().RemoveStreamHandler(gomock.Any()).AnyTimes()

			dummyCfg := &cfg.P2PConfig{}
			pm := &peerManager{
				peerFinder:   mockPeerFinder,
				wpManager:    mockWPManager,
				remotePeers:  make(map[types.PeerID]p2pcommon.RemotePeer, 10),
				waitingPeers: make(map[types.PeerID]*p2pcommon.WaitingPeer, 10),
				conf:         dummyCfg,
				nt:           mockNT,

				getPeerChannel:    make(chan getPeerTask),
				peerHandshaked:    make(chan handshakeResult),
				removePeerChannel: make(chan p2pcommon.RemotePeer),
				fillPoolChannel:   make(chan []p2pcommon.PeerMeta, 2),
				inboundConnChan:   make(chan inboundConnEvent),
				workDoneChannel:   make(chan p2pcommon.ConnWorkResult),
				eventListeners:    make([]p2pcommon.PeerEventListener, 0, 4),
				finishChannel:     make(chan struct{}),

				logger: logger,
			}

			go pm.runManagePeers()

			workWG := sync.WaitGroup{}
			workWG.Add(len(tt.conns))
			latch := sync.WaitGroup{}
			latch.Add(len(tt.conns))
			finCnt := uint32(0)
			for i, conn := range tt.conns {
				meta := p2pcommon.PeerMeta{ID: conn.pid, Outbound: conn.outbound}
				wr := p2pcommon.ConnWorkResult{Meta: meta, Result: nil, Inbound: !conn.outbound, Seq: uint32(i)}
				go func(conn desc, result p2pcommon.ConnWorkResult) {
					latch.Done()
					latch.Wait()
					//fmt.Printf("work start  %s #%d",p2putil.ShortForm(meta.ID),i)
					//time.Sleep(conn.hsTime)
					fmt.Printf("work done   %s #%d\n", p2putil.ShortForm(meta.ID), wr.Seq)
					pm.workDoneChannel <- result
				}(conn, wr)
			}
			mockWPManager.EXPECT().OnWorkDone(gomock.AssignableToTypeOf(p2pcommon.ConnWorkResult{})).Do(
				func(wr p2pcommon.ConnWorkResult) {
					atomic.AddUint32(&finCnt, 1)
					workWG.Done()
				}).AnyTimes()

			workWG.Wait()
			pm.Stop()

			if atomic.LoadUint32(&finCnt) != uint32(len(tt.conns)) {
				t.Errorf("finished count %v want %v", finCnt, len(tt.conns))
			}
		})
	}
}

func Test_peerManager_Stop(t *testing.T) {
	// check if Stop is working.
	tests := []struct {
		name string

		prevStatus int32

		wantStatus      int32
		wantSentChannel bool
	}{
		// never send to finish channel twice.
		{"TInitial", initial, stopping, false},
		{"TRunning", running, stopping, true},
		{"TStopping", stopping, stopping, false},
		{"TStopped", stopped, stopped, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := &peerManager{
				logger:        logger,
				finishChannel: make(chan struct{}, 1),
			}

			atomic.StoreInt32(&pm.status, tt.prevStatus)
			pm.Stop()

			if atomic.LoadInt32(&pm.status) != tt.wantStatus {
				t.Errorf("mansger status %v, want %v ", toMStatusName(atomic.LoadInt32(&pm.status)),
					toMStatusName(tt.wantStatus))
			}
			var sent bool
			timeout := time.NewTimer(time.Millisecond << 6)
			select {
			case <-pm.finishChannel:
				sent = true
			case <-timeout.C:
				sent = false
			}
			if sent != tt.wantSentChannel {
				t.Errorf("signal sent %v, want %v ", sent, tt.wantSentChannel)
			}
		})
	}
}

// It tests idempotent of Stop method
func Test_peerManager_StopInRun(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// check if Stop is working.
	tests := []struct {
		name string

		callCnt    int
		wantStatus int32
	}{
		{"TStopOnce", 1, stopped},
		{"TStopTwice", 2, stopped},
		{"TInStopping", 3, stopped},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockNT := p2pmock.NewMockNetworkTransport(ctrl)
			mockNT.EXPECT().AddStreamHandler(gomock.Any(), gomock.Any()).AnyTimes()
			mockNT.EXPECT().RemoveStreamHandler(gomock.Any()).AnyTimes()

			mockPeerFinder := p2pmock.NewMockPeerFinder(ctrl)
			mockWPManager := p2pmock.NewMockWaitingPeerManager(ctrl)

			pm := &peerManager{
				logger:     logger,
				nt:         mockNT,
				peerFinder: mockPeerFinder,
				wpManager:  mockWPManager,

				mutex:         &sync.Mutex{},
				finishChannel: make(chan struct{}),
			}
			go pm.runManagePeers()
			// wait status of pm is changed to running
			for atomic.LoadInt32(&pm.status) != running {
				time.Sleep(time.Millisecond)
			}
			// stopping will be done within one second if normal status
			checkTimer := time.NewTimer(time.Second >> 3)
			for i := 0; i < tt.callCnt; i++ {
				pm.Stop()
				time.Sleep(time.Millisecond << 6)
			}
			succ := false
			failedTimeout := time.NewTimer(time.Second * 5)

			// check if status changed
		VERIFYLOOP:
			for {
				select {
				case <-checkTimer.C:
					if atomic.LoadInt32(&pm.status) == tt.wantStatus {
						succ = true
						break VERIFYLOOP
					} else {
						checkTimer.Stop()
						checkTimer.Reset(time.Second)
					}
				case <-failedTimeout.C:
					break VERIFYLOOP
				}
			}
			if !succ {
				t.Errorf("mansger status %v, want %v within %v", toMStatusName(atomic.LoadInt32(&pm.status)),
					toMStatusName(tt.wantStatus), time.Second*5)
			}
		})
	}
}

func toMStatusName(status int32) string {
	switch status {
	case initial:
		return "initial"
	case running:
		return "running"
	case stopping:
		return "stopping"
	case stopped:
		return "stopped"
	default:
		return "(invalid)" + strconv.Itoa(int(status))
	}
}

func Test_peerManager_tryRegister(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	// id0 is in both designated peer and hidden peer
	desigIDs := make([]types.PeerID, 3)
	desigPeers := make(map[types.PeerID]p2pcommon.PeerMeta, 3)

	hiddenIDs := make([]types.PeerID, 3)
	hiddenPeers := make(map[types.PeerID]bool)

	for i := 0; i < 3; i++ {
		pkey, _, _ := crypto.GenerateKeyPair(crypto.Secp256k1, 256)
		pid, _ := types.IDFromPrivateKey(pkey)
		desigIDs[i] = pid
		desigPeers[pid] = p2pcommon.PeerMeta{ID: pid}
	}
	hiddenIDs[0] = desigIDs[0]
	hiddenPeers[desigIDs[0]] = true

	for i := 1; i < 3; i++ {
		pkey, _, _ := crypto.GenerateKeyPair(crypto.Secp256k1, 256)
		pid, _ := types.IDFromPrivateKey(pkey)
		hiddenIDs[i] = pid
		hiddenPeers[pid] = true
	}

	type args struct {
		outound bool
		status  *types.Status
	}
	tests := []struct {
		name string
		args args

		wantSucc   bool
		wantDesign bool
		wantHidden bool
	}{
		// add inbound peer
		{"TIn", args{false,
			dummyStatus(dummyPeerID, false)}, true, false, false},
		// add inbound designated peer
		{"TInDesignated", args{false,
			dummyStatus(desigIDs[1], false)}, true, true, false},
		// add inbound hidden peer
		{"TInHidden", args{false,
			dummyStatus(dummyPeerID, true)}, true, false, true},
		// add inbound peer (hidden in node config)
		{"TInHiddenInConf", args{false,
			dummyStatus(hiddenIDs[1], false)}, true, false, true},
		{"TInH&D", args{false,
			dummyStatus(hiddenIDs[0], true)}, true, true, true},

		// add outbound peer
		{"TOut", args{true,
			dummyStatus(dummyPeerID, false)}, true, false, false},
		// add outbound designated peer
		{"TOutDesignated", args{true,
			dummyStatus(desigIDs[1], false)}, true, true, false},
		// add outbound hidden peer
		{"TOutHidden", args{true,
			dummyStatus(dummyPeerID, true)}, true, false, true},
		// add outbound peer (hidden in node config)
		{"TOutHiddenInConf", args{true,
			dummyStatus(hiddenIDs[1], false)}, true, false, true},
		{"TOutH&D", args{true,
			dummyStatus(hiddenIDs[0], true)}, true, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStream := p2pmock.NewMockStream(ctrl)
			mockStream.EXPECT().Close().AnyTimes()
			mockRW := p2pmock.NewMockMsgReadWriter(ctrl)
			mockRW.EXPECT().ReadMsg().DoAndReturn(func() (interface{}, error) {
				time.Sleep(time.Millisecond * 10)
				return nil, errors.New("close")
			}).AnyTimes()
			mockPeerFactory := p2pmock.NewMockPeerFactory(ctrl)
			mockPeer := p2pmock.NewMockRemotePeer(ctrl)

			in := handshakeResult{meta: p2pcommon.NewMetaFromStatus(tt.args.status, tt.args.outound), status: tt.args.status, msgRW: mockRW, s: mockStream}
			var gotMeta p2pcommon.PeerMeta

			mockPeerFactory.EXPECT().CreateRemotePeer(gomock.AssignableToTypeOf(p2pcommon.PeerMeta{}), gomock.Any(), in.status, mockStream, mockRW).Do(func(meta p2pcommon.PeerMeta, seq uint32, status *types.Status, stream network.Stream, rw p2pcommon.MsgReadWriter) {
				gotMeta = meta
			}).Return(mockPeer)
			mockPeer.EXPECT().RunPeer().MaxTimes(1)
			mockPeer.EXPECT().Role().Return(p2pcommon.BlockProducer).AnyTimes()
			mockPeer.EXPECT().Name().Return("testPeer").AnyTimes()

			// in cases of handshake error
			mockMF := p2pmock.NewMockMoFactory(ctrl)
			mockMF.EXPECT().NewMsgRequestOrder(false, p2pcommon.GoAway, gomock.Any()).Return(&pbRequestOrder{}).MaxTimes(1)
			mockRW.EXPECT().WriteMsg(gomock.Any()).MaxTimes(1)

			pm := &peerManager{
				peerFactory:     mockPeerFactory,
				designatedPeers: desigPeers,
				hiddenPeerSet:   hiddenPeers,
				logger:          logger,
				mutex:           &sync.Mutex{},
				remotePeers:     make(map[types.PeerID]p2pcommon.RemotePeer, 100),
				peerHandshaked:  make(chan handshakeResult, 10),
			}


			r := pm.tryRegister(in)
			if (r != nil) != tt.wantSucc {
				t.Errorf("peerManager.tryRegister() succ = %v, want %v", r != nil, tt.wantSucc)
			}
			if tt.wantSucc {
				got := gotMeta
				if got.Designated != tt.wantDesign {
					t.Errorf("peerManager.tryRegister() got Designated = %v, want %v", got.Designated, tt.wantDesign)
				}
				if got.Hidden != tt.wantHidden {
					t.Errorf("peerManager.tryRegister() got Hidden = %v, want %v", got.Hidden, tt.wantHidden)
				}

			}
		})
	}
}

func Test_peerManager_tryRegisterCollision(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	selfID := p2pkey.NodeID()
	inboundWillLive := p2putil.ComparePeerID(selfID, dummyPeerID) <= 0
	type args struct {
		outbound bool
		status   *types.Status
	}
	tests := []struct {
		name string
		args args

		wantSucc bool
	}{
		// internal test self peerid is higher than test dummyPeerID
		{"TIn", args{false,
			dummyStatus(dummyPeerID, false)}, inboundWillLive},
		{"TOut", args{true,
			dummyStatus(dummyPeerID, false)}, !inboundWillLive},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockIS := p2pmock.NewMockInternalService(ctrl)
			mockStream := p2pmock.NewMockStream(ctrl)
			mockStream.EXPECT().Close().AnyTimes()
			mockRW := p2pmock.NewMockMsgReadWriter(ctrl)
			mockRW.EXPECT().ReadMsg().DoAndReturn(func() (interface{}, error) {
				time.Sleep(time.Millisecond * 10)
				return nil, errors.New("close")
			}).AnyTimes()

			in := handshakeResult{meta: p2pcommon.NewMetaFromStatus(tt.args.status, tt.args.outbound), status: tt.args.status, msgRW: mockRW, s: mockStream}
			mockIS.EXPECT().SelfNodeID().Return(selfID).MinTimes(1)
			mockPeerFactory := p2pmock.NewMockPeerFactory(ctrl)
			mockPeer := p2pmock.NewMockRemotePeer(ctrl)
			mockPeer.EXPECT().RunPeer().MaxTimes(1)
			mockPeer.EXPECT().Role().Return(p2pcommon.BlockProducer).AnyTimes()
			mockPeer.EXPECT().Name().Return("testPeer").AnyTimes()
			if tt.wantSucc {
				mockPeer.EXPECT().Stop().MaxTimes(1)
				mockPeerFactory.EXPECT().CreateRemotePeer(gomock.AssignableToTypeOf(p2pcommon.PeerMeta{}), gomock.Any(), in.status, mockStream, mockRW).Return(mockPeer)
			}

			// in cases of handshake error
			mockRW.EXPECT().WriteMsg(gomock.Any()).MaxTimes(1)

			pm := &peerManager{
				is: mockIS,
				peerFactory:     mockPeerFactory,
				designatedPeers: make(map[types.PeerID]p2pcommon.PeerMeta),
				hiddenPeerSet:   make(map[types.PeerID]bool),
				logger:          logger,
				mutex:           &sync.Mutex{},
				remotePeers:     make(map[types.PeerID]p2pcommon.RemotePeer, 100),
				peerHandshaked:  make(chan handshakeResult, 10),
			}
			pm.remotePeers[dummyPeerID] = mockPeer


			r := pm.tryRegister(in)
			if (r != nil) != tt.wantSucc {
				t.Errorf("peerManager.tryRegister() succ = %v, want %v", r != nil, tt.wantSucc)
			}
		})
	}
}
