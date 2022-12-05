package eth

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	ethtypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/sisu-network/lib/log"
)

func getSigner(client *ethclient.Client) ethtypes.Signer {
	chainId, err := client.ChainID(context.Background())
	if err != nil {
		panic(err)
	}

	return ethtypes.NewLondonSigner(chainId)
}

// TransferEth transfers a specific ETH amount to an address.
func TransferEth(client *ethclient.Client, mnemonic, chain string, recipient common.Address, amount *big.Int) error {
	_, account := getPrivateKey(mnemonic)
	log.Info("from address = ", account.String(), " to Address = ", recipient.String())

	nonce, err := client.PendingNonceAt(context.Background(), account)
	if err != nil {
		return fmt.Errorf("Failed to get nonce, err  = %s", err.Error())
	}

	gasPrice, err := client.SuggestGasPrice(context.Background())
	if err != nil {
		return err
	}
	if gasPrice.Cmp(big.NewInt(0)) <= 0 {
		return fmt.Errorf("Invalid gas price %s", gasPrice)
	}

	log.Info("Gas price = ", gasPrice, " on chain ", chain)

	gasLimit := uint64(22000) // in units
	amountFloat := new(big.Float).Quo(new(big.Float).SetInt(amount), new(big.Float).SetInt(ONE_ETHER_IN_WEI))
	log.Info("Amount in ETH: ", amountFloat, " on chain ", chain)

	var data []byte
	tx := ethtypes.NewTransaction(nonce, recipient, amount, gasLimit, gasPrice, data)

	privateKey, _ := getPrivateKey(mnemonic)
	signedTx, err := ethtypes.SignTx(tx, getSigner(client), privateKey)

	log.Info("Tx hash = ", signedTx.Hash(), " on chain ", chain)

	err = client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return fmt.Errorf("Failed to transfer ETH on chain %s, err = %s", chain, err)
	}

	bind.WaitDeployed(context.Background(), client, signedTx)

	return waitForTx(client, signedTx.Hash())
}

func waitForTx(client *ethclient.Client, hash common.Hash) error {
	start := time.Now()
	end := start.Add(time.Minute * 2)

	for {
		if time.Now().After(end) {
			return fmt.Errorf("Time out for transaction with hash %s", hash)
		}

		tx, isPending, err := client.TransactionByHash(context.Background(), hash)
		if err != nil && err != ethereum.NotFound {
			return fmt.Errorf("Failed to get transaction with hash %s", hash)
		}

		if tx == nil || isPending {
			time.Sleep(time.Second * 3)
			continue
		}

		break
	}

	return nil
}
