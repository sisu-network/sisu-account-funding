package core

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
	ethcommon "github.com/ethereum/go-ethereum/common"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"
	libchain "github.com/sisu-network/lib/chain"
	"github.com/sisu-network/sisu-account-funding/core/eth"
	"github.com/sisu-network/sisu-account-funding/core/types"
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

func getVault(chain string, vaults []*Vault) *Vault {
	for _, vault := range vaults {
		if vault.Chain == chain {
			return vault
		}
	}

	return nil
}

func getEcdsaPubkey(sisuRpc string) ethcommon.Address {
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

	pubKeyBytes := res.Pubkeys[libchain.KEY_TYPE_ECDSA]
	pubKey, err := ethcrypto.UnmarshalPubkey(pubKeyBytes)
	if err != nil {
		panic(err)
	}
	return ethcrypto.PubkeyToAddress(*pubKey)
}

func readMnemonic() string {
	fmt.Print("Enter mnemonic: ")
	reader := bufio.NewReader(os.Stdin)
	text, err := reader.ReadString('\n')
	if err != nil {
		panic(err)
	}

	return text
}

func Run() {
	mnemonic := readMnemonic()
	cfg := loadChainConfig("chains.toml")
	sisuRpc := "0.0.0.0:9090"

	for chain, chainCfg := range cfg.Chains {
		if libchain.IsETHBasedChain(chain) {
			sisuAccount := getEcdsaPubkey(sisuRpc)
			watcher := eth.NewWatcher(mnemonic, chain, chainCfg.Rpcs, sisuAccount.String())
			watcher.Start()
			break
		}
	}
}
