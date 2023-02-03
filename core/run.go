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
	pubkeys := getPubkeys("0.0.0.0:9090")

	for chain, chainCfg := range cfg.Chains {
		if libchain.IsETHBasedChain(chain) {
			// 04cbf8c5562928f81495a55c12f89836cf744c1adcc43c45c33e97571603b979bcead13fb7f33e7cc01d2a632337725b84942d3998c933b08e5a34be8df7794e05
			// is the test ecdsa pubkey. Use hex.DecodeString to get its bytes
			sisuAccount := getEthAccount(pubkeys)
			watcher := eth.NewWatcher(mnemonic, chain, chainCfg.Rpcs, sisuAccount.String())
			watcher.Start()
		}

		// Use 7cbb424e0dffad3104e29c6febe3abd899b2d2b972475dabd9fbe6b62f9af2ff as hex of sample test
		// eddsa pubkey. Use hex.DecodeString to get its bytes
		if libchain.IsLiskChain(chain) {
			// edPubkey, _ := hex.DecodeString("7cbb424e0dffad3104e29c6febe3abd899b2d2b972475dabd9fbe6b62f9af2ff")
			edPubkey := pubkeys[libchain.KEY_TYPE_EDDSA]
			watcher := lisk.NewWatcher(mnemonic, chain, chainCfg.Rpcs[0], edPubkey)
			watcher.Start()
		}
	}
}
