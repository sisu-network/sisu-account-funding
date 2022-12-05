package core

import (
	"context"

	"github.com/sisu-network/sisu-account-funding/core/types"
	"google.golang.org/grpc"
)

func GetAllPubkeys(sisuRpc string) (*types.QueryAllPubKeysResponse, error) {
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

	return res, err
}
