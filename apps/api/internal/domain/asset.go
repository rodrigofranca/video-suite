package domain

import "time"

type Asset struct {
	ID        string
	Filename  string
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}
