package provider

import (
	"context"
	"fmt"

	"github.com/brucexc/pray-to-earn/internal/config"
	"github.com/brucexc/pray-to-earn/internal/database"
)

func ProvideDatabaseClient(configFile *config.File) (*database.Client, error) {
	databaseClient, err := database.Dial(context.TODO(), configFile.Database.URI)
	if err != nil {
		return nil, fmt.Errorf("dial to database: %w", err)
	}

	if err = databaseClient.Migrate(context.TODO()); err != nil {
		return nil, fmt.Errorf("mrigate database: %w", err)
	}

	return databaseClient, nil
}
