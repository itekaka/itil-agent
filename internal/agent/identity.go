package agent

import (
	"os"
	"strings"

	"github.com/google/uuid"
)

func LoadOrCreateUUID(path string) (string, error) {
	if b, err := os.ReadFile(path); err == nil {
		id := strings.TrimSpace(string(b))
		if id != "" {
			return id, nil
		}
	}
	id := uuid.New().String()
	if err := os.WriteFile(path, []byte(id), 0600); err != nil {
		return "", err
	}
	return id, nil
}
