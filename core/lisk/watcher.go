package lisk

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"reflect"
	"strconv"
	"time"

	"go.uber.org/atomic"
	"google.golang.org/protobuf/proto"

	deyeslisk "github.com/sisu-network/deyes/chains/lisk"
	liskcrypto "github.com/sisu-network/deyes/chains/lisk/crypto"
	lisktypes "github.com/sisu-network/deyes/chains/lisk/types"
	"github.com/sisu-network/deyes/config"

	"github.com/sisu-network/lib/log"
)

var (
	SleepTime     = time.Second * 60 * 30
	FundThreshold = new(big.Int).SetInt64(10 * 100_000_000)
	FundAmount    = uint64(10 * 100_000_000)
)

type watcher struct {
	chain     string
	mnemonic  string
	url       string
	pubkey    []byte
	watchAddr string
	stop      atomic.Bool
}

func NewWatcher(mnemonic string, chain string, url string, pubkey []byte) *watcher {
	fmt.Println("Lisk32 = ", liskcrypto.GetLisk32AddressFromPublickey(pubkey))
	return &watcher{
		mnemonic:  mnemonic,
		chain:     chain,
		url:       url,
		pubkey:    pubkey,
		watchAddr: liskcrypto.GetLisk32AddressFromPublickey(pubkey),
		stop:      *atomic.NewBool(false),
	}
}

func (w *watcher) Start() {
	log.Infof("Starting watcher for chain %s, watch address = %s", w.chain, w.watchAddr)
	go w.loop()
}

func (w *watcher) Stop() {
	w.stop.Store(true)
}

func (w *watcher) loop() {
	for {
		if w.stop.Load() {
			break
		}

		bz, err := w.get("/accounts", map[string]string{
			"address": w.watchAddr,
			"limit":   "10",
			"offset":  "0",
		})

		if err != nil {
			log.Errorf("Cannot get accounts info, err = ", err)
		} else {
			res := &GetAccountsResponse{}
			err = json.Unmarshal(bz, res)
			if err != nil {
				log.Errorf("Failed to get unmarshal get accounts response, err = ", err)
			} else {
				if len(res.Data) > 0 {
					balance, ok := new(big.Int).SetString(res.Data[0].Summary.Balance, 10)
					if ok {
						if balance.Cmp(FundThreshold) < 0 {
							w.fundSisu(w.mnemonic, w.pubkey, FundAmount, "")
						}
					}
				}
			}
		}

		time.Sleep(SleepTime)
	}
}

func (w *watcher) get(endpoint string, params map[string]string) ([]byte, error) {
	keys := reflect.ValueOf(params).MapKeys()
	req, err := http.NewRequest("GET", w.url+endpoint, nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	for _, key := range keys {
		q.Add(key.Interface().(string), params[key.Interface().(string)])
	}

	req.URL.RawQuery = q.Encode()
	response, err := http.Get(req.URL.String())
	if response == nil {
		return nil, fmt.Errorf("cannot fetch data " + endpoint)
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	return responseData, err
}

func (w *watcher) fundSisu(mnemonic string, mpcPubKey []byte, amount uint64, data string) error {
	log.Info("Funding sisu....")
	deyesChainCfg := config.Chain{Chain: w.chain, Rpcs: []string{w.url}}

	client := deyeslisk.NewLiskClient(deyesChainCfg)
	mpcAddr := liskcrypto.GetAddressFromPublicKey(mpcPubKey)
	log.Verbose("Funding LSK for mpc address = ", mpcAddr)

	receiver := mpcAddr

	moduleId := uint32(2)
	assetId := uint32(0)

	privateKey := liskcrypto.GetPrivateKeyFromSecret(mnemonic)
	faucetPubKey := liskcrypto.GetPublicKeyFromSecret(mnemonic)

	lisk32 := liskcrypto.GetLisk32AddressFromPublickey(faucetPubKey)
	log.Verbosef("Lisk32 of the faucet = %s", lisk32)
	acc, err := client.GetAccount(lisk32)
	if err != nil {
		panic(err)
	}

	nonce, err := strconv.ParseUint(acc.Sequence.Nonce, 10, 64)
	if err != nil {
		panic(err)
	}

	recipientAddress, err := hex.DecodeString(receiver)
	if err != nil {
		panic(err)
	}

	fee := uint64(500_000)
	assetPb := &lisktypes.AssetMessage{
		Amount:           &amount,
		RecipientAddress: recipientAddress,
		Data:             &data,
	}

	asset, err := proto.Marshal(assetPb)
	tx := &lisktypes.TransactionMessage{
		ModuleID:        &moduleId,
		AssetID:         &assetId,
		Fee:             &fee,
		Asset:           asset,
		Nonce:           &nonce,
		SenderPublicKey: faucetPubKey,
	}
	bz, err := proto.Marshal(tx)
	if err != nil {
		panic(err)
	}

	bytesToSign, err := liskcrypto.GetSigningBytes(lisktypes.NetworkId[w.chain], bz)
	if err != nil {
		return fmt.Errorf("Failed to get lisk bytes to sign, err = %s", err)
	}

	signature := liskcrypto.SignMessage(bytesToSign, privateKey)
	tx.Signatures = [][]byte{signature}
	signedBz, err := proto.Marshal(tx)
	if err != nil {
		panic(err)
	}

	hash := sha256.Sum256(signedBz)
	log.Verbosef("Calculated hash = %s", hex.EncodeToString(hash[:]))
	log.Infof("Funding Sisu from account %s to account %s= ", lisk32,
		liskcrypto.GetLisk32AddressFromPublickey(mpcPubKey))

	txHash, err := client.CreateTransaction(hex.EncodeToString(signedBz))
	if err != nil {
		panic(err)
	}

	log.Info("Lisk txHash = ", txHash)

	return nil
}
