package schema

import "github.com/ethereum/go-ethereum/common"

type Note struct {
	ID        uint64         `json:"id"`
	Address   common.Address `json:"address"`
	Note      string         `json:"note"`
	CreatedAt int64          `json:"created_at"`
}
