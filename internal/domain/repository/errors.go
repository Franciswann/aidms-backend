// Package repository declares the persistence and runtime contracts the
// Use Case layer depends on. Concrete implementations live in
// internal/repository (GORM) and internal/docker (Docker SDK).
package repository

import "errors"

// ErrNotFound is returned by repository implementations when a record
// does not exist, so callers in the Use Case layer never need to know
// about the underlying storage driver's error type (e.g. GORM).
var ErrNotFound = errors.New("record not found")
