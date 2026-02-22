package tool

import "github.com/google/uuid"

func GenerateUUIDV7() string {
	return uuid.Must(uuid.NewV7()).String()
}
