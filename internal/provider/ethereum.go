package provider

import (
	"context"
	"fmt"

	"github.com/brucexc/pray-to-earn/internal/config"
	"github.com/ethereum/go-ethereum/ethclient"
)

func ProvideEthereumClient(configFile *config.File) (*ethclient.Client, error) {
	ethereumClient, err := ethclient.DialContext(context.TODO(), configFile.RSS3Chain.Endpoint)
	if err != nil {
		return nil, fmt.Errorf("dial to endpoint: %w", err)
	}

	return ethereumClient, nil
}
