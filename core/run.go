package core

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"syscall"

	"github.com/BurntSushi/toml"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	libchain "github.com/sisu-network/lib/chain"
	"github.com/sisu-network/sisu-account-funding/core/eth"
	"github.com/sisu-network/sisu-account-funding/core/lisk"
	"github.com/sisu-network/sisu-account-funding/core/types"
	"golang.org/x/term"
	"google.golang.org/grpc"
)

func loadChainConfig(filePath string) *ChainsCfg {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		panic(err)
	}

	cfg := new(ChainsCfg)
	_, err := toml.DecodeFile(filePath, &cfg)
	if err != nil {
		panic(err)
	}

	return cfg
}

func loadVaults(filePath string) ([]*Vault, error) {
	cfg := make([]*Vault, 0)

	dat, err := os.ReadFile(filePath)
	if err != nil {
		return cfg, err
	}

	if err := json.Unmarshal(dat, &cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

func getPubkeys(sisuRpc string) map[string][]byte {
	grpcConn, err := grpc.Dial(
		sisuRpc,
		grpc.WithInsecure(),
	)
	defer grpcConn.Close()
	if err != nil {
		panic(err)
	}

	queryClient := types.NewTssQueryClient(grpcConn)

	res, err := queryClient.AllPubKeys(context.Background(), &types.QueryAllPubKeysRequest{})
	if err != nil {
		panic(err)
	}

	return res.Pubkeys
}

func getEthAccount(pubkeys map[string][]byte) ethcommon.Address {
	pubKeyBytes := pubkeys[libchain.KEY_TYPE_ECDSA]
	pubKey, err := ethcrypto.UnmarshalPubkey(pubKeyBytes)
	if err != nil {
		panic(err)
	}

	return ethcrypto.PubkeyToAddress(*pubKey)
}

func readMnemonic() string {
	fmt.Print("Enter mnemonic: ")

	bytePassword, err := term.ReadPassword(int(syscall.Stdin))
	if err != nil {
		panic(err)
	}

	return string(bytePassword)
}

func Run() {
	mnemonic := readMnemonic()
	cfg := loadChainConfig("chains.toml")
	sisuRpc := "0.0.0.0:9090"

	pubkeys := getPubkeys(sisuRpc)

	for chain, chainCfg := range cfg.Chains {
		if libchain.IsETHBasedChain(chain) {
			sisuAccount := getEthAccount(pubkeys)
			watcher := eth.NewWatcher(mnemonic, chain, chainCfg.Rpcs, sisuAccount.String())
			watcher.Start()
		}

		if libchain.IsLiskChain(chain) {
			watcher := lisk.NewWatcher(mnemonic, chain, chainCfg.Rpcs[0], pubkeys[libchain.KEY_TYPE_EDDSA])
			watcher.Start()
		}
	}
}
