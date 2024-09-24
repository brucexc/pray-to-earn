package table

import (
	"time"

	"github.com/brucexc/pray-to-earn/schema"
	"github.com/ethereum/go-ethereum/common"
)

type Note struct {
	ID        uint64         `gorm:"column:id;primaryKey"`
	Address   common.Address `gorm:"column:address"`
	Note      string         `gorm:"column:note"`
	CreatedAt time.Time      `gorm:"column:created_at"`
}

func (n *Note) TableName() string {
	return "note"
}

func (n *Note) Import(note *schema.Note) error {
	n.ID = note.ID
	n.Address = note.Address
	n.Note = note.Note

	return nil
}

func (n *Note) Export() (*schema.Note, error) {
	return &schema.Note{
		ID:        n.ID,
		Address:   n.Address,
		Note:      n.Note,
		CreatedAt: n.CreatedAt.Unix(),
	}, nil
}
