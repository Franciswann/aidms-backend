package entity

import "time"

type File struct {
	ID        string
	UserID    string
	Name      string
	Path      string
	MimeType  string
	Size      int64
	CreatedAt time.Time
	UpdatedAt time.Time
}
