package instance

import (
	"crypto/rand"
	"fmt"
)

// UUID represents a unique identifier.
type UUID string

// NewUUID generates a new UUID v4.
func NewUUID() (UUID, error) {
	var uuid [16]byte
	_, err := rand.Read(uuid[:])
	if err != nil {
		return "", fmt.Errorf("failed to generate UUID: %w", err)
	}

	// Set version 4 (random) and variant bits
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return UUID(fmt.Sprintf("%x-%x-%x-%x-%x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])), nil
}

// String returns the string representation of the UUID.
func (u UUID) String() string {
	return string(u)
}