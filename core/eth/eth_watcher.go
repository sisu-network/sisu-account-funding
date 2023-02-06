package eth

import (
	"context"
	"math/big"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sisu-network/lib/log"
	"go.uber.org/atomic"
)

var (
	SleepTime        = time.Second * 60 * 30
	ONE_ETHER_IN_WEI = big.NewInt(1000000000000000000)
)

type watcher struct {
	mnemonic  string
	chain     string
	urls      []string
	clients   []*ethclient.Client
	watchAddr ethcommon.Address
	stop      atomic.Bool
}

func NewWatcher(mnemonic string, chain string, urls []string, watchAddr string) *watcher {
	return &watcher{
		mnemonic:  mnemonic,
		chain:     chain,
		urls:      urls,
		clients:   make([]*ethclient.Client, len(urls)),
		watchAddr: ethcommon.HexToAddress(watchAddr),
		stop:      *atomic.NewBool(false),
	}
}

func (w *watcher) Start() {
	log.Infof("Starting watcher for chain %s, watch address = %s",
		w.chain, w.watchAddr.String())
	w.init()

	go w.loop()
}

func (w *watcher) Stop() {
	w.stop.Store(true)
}

func (w *watcher) init() {
	for i, url := range w.urls {
		client, err := ethclient.Dial(url)
		if err != nil {
			log.Error("cannot dial source chain, url = ", url)
			continue
		}

		log.Verbosef("Setting client for url %s", url)
		w.clients[i] = client
	}
}

func (w *watcher) loop() {
	threshold := new(big.Int).Div(ONE_ETHER_IN_WEI, big.NewInt(10))
	fundingAmount := big.NewInt(30_000_000_000_000_000) // 0.03 ETH

	for {
		if w.stop.Load() {
			return
		}

		// Query the account balance.
		for i, client := range w.clients {
			balance, err := client.BalanceAt(context.Background(), w.watchAddr, nil)
			if err != nil {
				log.Errorf("Failed to get balance on chain %s, url = %s, err = %s", w.chain, w.urls[i], err.Error())
				continue
			}

			amountFloat := new(big.Float).Quo(new(big.Float).SetInt(balance), new(big.Float).SetInt(ONE_ETHER_IN_WEI))
			log.Verbose("Amount in ETH: ", amountFloat, " on chain ", w.chain)

			if balance.Cmp(threshold) < 0 {
				log.Infof("Funding chain %s", w.chain)
				// Balance is less than the threshold. Let's top up the account.
				err := TransferEth(client, w.mnemonic, w.chain, w.watchAddr, fundingAmount)
				if err != nil {
					log.Errorf("Failed to transfer eth on chain %s, err  = %s", w.chain, err.Error())
				}
			}

			break
		}

		time.Sleep(SleepTime)
	}
}
