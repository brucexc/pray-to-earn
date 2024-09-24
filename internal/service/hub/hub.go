package hub

import (
	"context"
	"fmt"
	"github.com/brucexc/pray-to-earn/contract"
	"github.com/brucexc/pray-to-earn/contract/pray"
	"github.com/brucexc/pray-to-earn/internal/config"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/redis/go-redis/v9"
	"math/big"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

type Hub struct {
	//databaseClient *database.Client
	prayContract   *pray.Pray
	auth           *bind.TransactOpts
	ethereumClient *ethclient.Client
	redisClient    *redis.Client
}

var _ echo.Validator = (*Validator)(nil)

var defaultValidator = &Validator{
	validate: validator.New(),
}

type Validator struct {
	validate *validator.Validate
}

func (v *Validator) Validate(i interface{}) error {
	return v.validate.Struct(i)
}

func NewHub(ctx context.Context, conf config.File, ethereumClient *ethclient.Client, redisClient *redis.Client) (*Hub, error) {
	prayContract, err := pray.NewPray(contract.AddressPray, ethereumClient)
	if err != nil {
		return nil, fmt.Errorf("new pray contract: %w", err)
	}

	privateKey, _ := crypto.HexToECDSA(conf.AdminKey)

	auth, _ := bind.NewKeyedTransactorWithChainID(privateKey, big.NewInt(2331))

	auth.GasPrice, _ = ethereumClient.SuggestGasPrice(ctx)

	auth.GasLimit = uint64(300000)

	return &Hub{
		redisClient:    redisClient,
		prayContract:   prayContract,
		auth:           auth,
		ethereumClient: ethereumClient,
	}, nil
}
