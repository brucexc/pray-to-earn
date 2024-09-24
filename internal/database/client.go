package database

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"github.com/brucexc/pray-to-earn/internal/database/table"
	"github.com/brucexc/pray-to-earn/schema"
	"github.com/pressly/goose/v3"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"moul.io/zapgorm2"
)

//go:embed migration/*.sql
var migrationFS embed.FS

var (
	ErrorRowNotFound = errors.New("row not found")
)

type Client struct {
	database *gorm.DB
}

func (c *Client) Migrate(ctx context.Context) error {
	goose.SetBaseFS(migrationFS)
	goose.SetTableName("versions")

	if err := goose.SetDialect(new(postgres.Dialector).Name()); err != nil {
		return fmt.Errorf("set migration dialect: %w", err)
	}

	connector, err := c.database.DB()
	if err != nil {
		return fmt.Errorf("get database connector: %w", err)
	}

	return goose.UpContext(ctx, connector, "migration")
}

func (c *Client) GetNote(ctx context.Context, id uint64) (*schema.Note, error) {
	var note table.Note

	if err := c.database.WithContext(ctx).First(&note, "id = ?", id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrorRowNotFound
		}

		return nil, err
	}

	return note.Export()
}

func (c *Client) SaveNote(ctx context.Context, data *schema.Note) error {
	var note table.Note

	if err := note.Import(data); err != nil {
		return err
	}

	return c.database.WithContext(ctx).Create(&note).Error
}

func Dial(_ context.Context, dataSourceName string) (*Client, error) {
	logger := zapgorm2.New(zap.L())
	logger.SetAsDefault()

	config := gorm.Config{
		Logger: logger,
	}

	databaseClient, err := gorm.Open(postgres.Open(dataSourceName), &config)
	if err != nil {
		return nil, fmt.Errorf("dial database: %w", err)
	}

	return &Client{
		database: databaseClient,
	}, nil
}
