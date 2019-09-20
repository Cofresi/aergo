/*
 * @file
 * @copyright defined in aergo/LICENSE.txt
 */

package server

import (
	"github.com/Cofresi/aergo-lib/log"
	"github.com/Cofresi/aergo/config"
	"github.com/Cofresi/aergo/contract/enterprise"
	"github.com/Cofresi/aergo/p2p/p2putil"
	"github.com/Cofresi/aergo/types"
	"github.com/golang/mock/gomock"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

const testAuthDir = "/tmp"

var sampleEntries []enterprise.WhiteListEntry
func init() {
	eIDIP, _ := enterprise.NewWhiteListEntry(`{"peerid":"16Uiu2HAmPZE7gT1hF2bjpg1UVH65xyNUbBVRf3mBFBJpz3tgLGGt", "address":"172.21.3.35" }`)
	eIDIR, _ := enterprise.NewWhiteListEntry(`{"peerid":"16Uiu2HAmN5YU8V2LnTy9neuuJCLNsxLnd5xVSRZqkjvZUHS3mLoD", "cidr":"172.21.3.35/16" }`)
	eID, _ := enterprise.NewWhiteListEntry(`{"peerid":"16Uiu2HAkvvhjxVm2WE9yFBDdPQ9qx6pX9taF6TTwDNHs8VPi1EeR" }`)
	eIR, _ := enterprise.NewWhiteListEntry(`{"cidr":"211.5.3.123/16" }`)
	eIP6, _ := enterprise.NewWhiteListEntry(`{"address":"2001:0db8:0123:4567:89ab:cdef:1234:5678" }`)
	eIR6, _ := enterprise.NewWhiteListEntry(`{"cidr":"2001:0db8:0123:4567:89ab:cdef:1234:5678/96" }`)
	sampleEntries = []enterprise.WhiteListEntry{eIDIP, eIDIR, eID, eIR, eIP6, eIR6}
}
func Test_polarisListManager_saveListFile(t *testing.T) {

	logger := log.NewLogger("polaris.test")
	conf := config.PolarisConfig{EnableBlacklist: true}
	lm := NewPolarisListManager(&conf, testAuthDir, logger)
	lm.entries = sampleEntries
	lm.saveListFile()
	defer func() {
		os.Remove(filepath.Join(testAuthDir, localListFile))
	}()

	lm2 := NewPolarisListManager(&conf, testAuthDir, logger)
	lm2.loadListFile()
	if len(lm2.entries) != len(lm.entries) {
		t.Errorf("polarisListManager.loadListFile() entry count %v, want %v", len(lm2.entries), len(lm.entries))
	}

	for i, e := range lm.entries {
		e2 := lm.entries[i]

		if !reflect.DeepEqual(e, e2) {
			t.Errorf("polarisListManager.loadListFile() entry %v, %v", e, e2)
		}
	}
}

func Test_polarisListManager_RemoveEntry(t *testing.T) {
	logger := log.NewLogger("polaris.test")
	conf := &config.PolarisConfig{EnableBlacklist: true}

	tests := []struct {
		name string
		idx  int
		want bool
	}{
		{"TFirst",0, true},
		{"TMid",1, true},
		{"TLast",5, true},
		{"TOverflow",6, false},
		{"TNegative",-1, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := NewPolarisListManager(conf, "temp", logger)
			lm.entries = sampleEntries
			if got := lm.RemoveEntry(tt.idx); got != tt.want {
				t.Errorf("RemoveEntry() = %v, want %v", got, tt.want)
			} else if got != (len(lm.entries) < 6) {
				t.Errorf("RemoveEntry() remain size = %v, want not", len(lm.entries))
			}

		})
	}
}

func Test_polarisListManager_AddEntry(t *testing.T) {
	logger := log.NewLogger("polaris.test")
	conf := &config.PolarisConfig{EnableBlacklist: true}

	tests := []struct {
		name string
		args []enterprise.WhiteListEntry
		wantSize int
	}{
		{"TFirst",sampleEntries[:0], 0},
		{"T1",sampleEntries[:1], 1},
		{"TAll",sampleEntries, 6},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := NewPolarisListManager(conf, "temp", logger)
			for _, e := range tt.args {
				lm.AddEntry(e)
			}

			if len(lm.ListEntries()) != tt.wantSize {
				t.Errorf("AddEntry() entries size = %v, want %v", len(lm.ListEntries()), tt.wantSize)
			}
		})
	}
}



func Test_polarisListManager_IsBanned(t *testing.T) {
	conf := config.NewServerContext("", "").GetDefaultPolarisConfig()
	conf.EnableBlacklist = true
	logger := log.NewLogger("polaris.test")

	addr1 := "123.45.67.89"
	id1 := p2putil.RandomPeerID()
	addrother := "8.8.8.8"
	idother := p2putil.RandomPeerID()
	thirdAddr := "222.8.8.8"
	thirdID := p2putil.RandomPeerID()

	IDOnly, e1 := enterprise.NewWhiteListEntry(`{"peerid":"`+id1.Pretty()+`"}`)
	AddrOnly, e2 := enterprise.NewWhiteListEntry(`{"address":"`+addr1+`"}`)
	IDAddr, e3 := enterprise.NewWhiteListEntry(`{"peerid":"`+idother.Pretty()+`", "address":"`+addrother+`"}`)
	if e1 !=nil || e2 != nil || e3 != nil {
		t.Fatalf("Inital entry value failure %v , %v , %v",e1,e2,e3)
	}
	listCfg := []enterprise.WhiteListEntry{IDOnly, AddrOnly, IDAddr}
	emptyCfg := []enterprise.WhiteListEntry{}


	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	type args struct {
		addr string
		pid  types.PeerID
	}
	tests := []struct {
		name string
		preset []enterprise.WhiteListEntry
		args args
		want bool
	}{
		{"TFoundBoth", listCfg, args{addr1, id1}, true},
		{"TIDOnly", listCfg, args{addrother, id1}, true},
		{"TIDOnly2", listCfg, args{thirdAddr, id1}, true},
		{"TIDOnlyFail", listCfg, args{thirdAddr, idother}, false},
		{"TAddrOnly1", listCfg, args{addr1, idother}, true},
		{"TAddrOnly2", listCfg, args{addr1, thirdID}, true},
		{"TIDAddrSucc", listCfg, args{addrother, idother}, true},
		{"TIDAddrFail", listCfg, args{addrother, thirdID}, false},
		{"TIDAddrFail2", listCfg, args{thirdAddr, idother}, false},

		// if config have nothing. everything is allowed
		{"TEmpFoundBoth", emptyCfg, args{addr1, id1}, false},
		{"TEmpIDOnly", emptyCfg, args{addrother, id1}, false},
		{"TEmpIDOnly2", emptyCfg, args{thirdAddr, id1}, false},
		{"TEmpIDOnly2", emptyCfg, args{thirdAddr, id1}, false},
		{"TEmpAddrOnly1", emptyCfg, args{addr1, idother}, false},
		{"TEmpAddrOnly2", emptyCfg, args{addr1, thirdID}, false},
		{"TEmpIDAddrSucc", emptyCfg, args{addrother, idother}, false},
		{"TEmpIDAddrFail", emptyCfg, args{addrother, id1}, false},
		{"TEmpIDAddrFail2", emptyCfg, args{thirdAddr, idother}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := NewPolarisListManager(conf, "/tmp", logger)
			b.entries = tt.preset

			if got, _ := b.IsBanned(tt.args.addr, tt.args.pid); got != tt.want {
				t.Errorf("listManagerImpl.IsBanned() = %v, want %v", got, tt.want)
			}
		})
	}
}
