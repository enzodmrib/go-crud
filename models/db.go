package models

import "github.com/google/uuid"

type DB[T any] map[uuid.UUID]T
