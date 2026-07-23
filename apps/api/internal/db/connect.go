package db

import (
	"context"

	"github.com/jackc/pgx/v5"
)

func Connect(databaseURL string) (*pgx.Conn, error) {
	return pgx.Connect(context.Background(), databaseURL)
}
