package key

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"path"
	"sync"
	"time"

	"github.com/Cofresi/aergo-lib/db"
	"github.com/aergoio/aergo/types"
	"github.com/btcsuite/btcd/btcec"
	sha256 "github.com/minio/sha256-simd"
)

type aergokey = btcec.PrivateKey

type keyPair struct {
	key   *aergokey
	timer *time.Timer
}

// Store stucture of keystore
type Store struct {
	sync.RWMutex
	timeout      time.Duration
	unlocked     map[string]*keyPair
	unlockedLock *sync.Mutex
	storage      db.DB
}

// NewStore make new instance of keystore
func NewStore(storePath string, unlockTimeout uint) *Store {
	const dbName = "account"
	dbPath := path.Join(storePath, dbName)
	return &Store{
		timeout:      time.Duration(unlockTimeout) * time.Second,
		unlocked:     map[string]*keyPair{},
		unlockedLock: &sync.Mutex{},
		storage:      db.NewDB(db.LevelImpl, dbPath),
	}
}
func (ks *Store) CloseStore() {
	ks.unlocked = nil
	ks.storage.Close()
}

// CreateKey make new key in keystore and return it's address
func (ks *Store) CreateKey(pass string) (Address, error) {
	//gen new key
	privkey, err := btcec.NewPrivateKey(btcec.S256())
	if err != nil {
		return nil, err
	}
	return ks.addKey(privkey, pass)
}

//ImportKey is to import encrypted key
func (ks *Store) ImportKey(imported []byte, oldpass string, newpass string) (Address, error) {
	hash := hashBytes([]byte(oldpass), nil)
	rehash := hashBytes([]byte(oldpass), hash)
	key, err := decrypt(hash, rehash, imported)
	if err != nil {
		return nil, err
	}
	privkey, pubkey := btcec.PrivKeyFromBytes(btcec.S256(), key)
	address := GenerateAddress(pubkey.ToECDSA())
	addresses, err := ks.GetAddresses()
	if err != nil {
		return nil, err
	}
	for _, v := range addresses {
		if bytes.Equal(address, v) {
			return nil, errors.New("already exist")
		}
	}
	err = ks.SaveAddress(address)
	if err != nil {
		return nil, err
	}
	return ks.addKey(privkey, newpass)
}

//ExportKey is to export encrypted key
func (ks *Store) ExportKey(addr Address, pass string) ([]byte, error) {
	key, err := ks.getKey(addr, pass)
	if key == nil {
		return nil, err
	}
	return EncryptKey(key, pass)
}

// EncryptKey encrypts a key with a given export for exporting
func EncryptKey(key []byte, pass string) ([]byte, error) {
	hash := hashBytes([]byte(pass), nil)
	rehash := hashBytes([]byte(pass), hash)
	return encrypt(hash, rehash, key)
}

//Unlock is to unlock account for signing
func (ks *Store) Unlock(addr Address, pass string) (Address, error) {
	key, err := ks.getKey(addr, pass)
	if key == nil {
		return nil, err
	}
	pk, _ := btcec.PrivKeyFromBytes(btcec.S256(), key)
	addrKey := types.EncodeAddress(addr)

	ks.unlockedLock.Lock()
	defer ks.unlockedLock.Unlock()

	unlockedKeyPair, exist := ks.unlocked[addrKey]

	if ks.timeout == 0 {
		ks.unlocked[addrKey] = &keyPair{key: pk, timer: nil}
		return addr, nil
	}

	if exist {
		unlockedKeyPair.timer.Reset(ks.timeout)
	} else {
		lockTimer := time.AfterFunc(ks.timeout,
			func() {
				ks.Lock(addr, pass)
			},
		)
		ks.unlocked[addrKey] = &keyPair{key: pk, timer: lockTimer}
	}
	return addr, nil
}

//Lock is to lock account prevent signing
func (ks *Store) Lock(addr Address, pass string) (Address, error) {
	key, err := ks.getKey(addr, pass)
	if key == nil {
		return nil, err
	}
	b58addr := types.EncodeAddress(addr)

	ks.unlockedLock.Lock()
	defer ks.unlockedLock.Unlock()

	if _, exist := ks.unlocked[b58addr]; exist {
		ks.unlocked[b58addr] = nil
		delete(ks.unlocked, b58addr)
	}
	return addr, nil
}

func (ks *Store) getKey(address []byte, pass string) ([]byte, error) {
	encryptkey := hashBytes(address, []byte(pass))
	key := ks.storage.Get(hashBytes(address, encryptkey))
	if cap(key) == 0 {
		return nil, types.ErrWrongAddressOrPassWord
	}
	return decrypt(address, encryptkey, key)
}

func (ks *Store) addKey(key *btcec.PrivateKey, pass string) (Address, error) {
	//gen new address
	address := GenerateAddress(&key.PublicKey)
	//save pass/address/key
	encryptkey := hashBytes(address, []byte(pass))
	encrypted, err := encrypt(address, encryptkey, key.Serialize())
	if err != nil {
		return nil, err
	}
	ks.storage.Set(hashBytes(address, encryptkey), encrypted)
	return address, nil
}

func hashBytes(b1 []byte, b2 []byte) []byte {
	h := sha256.New()
	h.Write(b1)
	h.Write(b2)
	return h.Sum(nil)
}

func encrypt(base, key, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	// Never use more than 2^32 random nonces with a given key because of the risk of a repeat.
	if len(base) < 16 {
		return nil, errors.New("too short address length")
	}
	nonce := base[4:16]

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	cipherbytes := aesgcm.Seal(nil, nonce, data, nil)
	return cipherbytes, nil
}

func decrypt(base, key, data []byte) ([]byte, error) {
	if len(base) < 16 {
		return nil, errors.New("too short address length")
	}
	nonce := base[4:16]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plainbytes, err := aesgcm.Open(nil, nonce, data, nil)

	if err != nil {
		return nil, err
	}
	return plainbytes, nil
}
