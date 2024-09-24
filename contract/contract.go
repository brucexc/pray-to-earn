package contract

import (
	"github.com/ethereum/go-ethereum/common"
)

//go:generate go run --mod=mod github.com/ethereum/go-ethereum/cmd/abigen@v1.13.5 --abi ./abi/Pray.abi --pkg pray --type Pray --out ./pray/pray.go

var (
	AddressPray      = common.HexToAddress("0xE26CFDE633A7be6714e58b44F2eA5Af8Ef080378") // https://scan.testnet.rss3.io/address/0xE26CFDE633A7be6714e58b44F2eA5Af8Ef080378
	TransferEventSig = common.HexToHash("0xddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
)
