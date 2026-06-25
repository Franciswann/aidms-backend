package entity

import "time"

type User struct {
	ID             string
	Email          string
	HashedPassword string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
