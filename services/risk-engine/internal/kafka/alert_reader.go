package kafka

import (
	"fmt"
	"strings"
)

func ValidateTopic(topic string) error {
	if strings.TrimSpace(topic) == "" {
		return fmt.Errorf("topic cannot be empty")
	}
	return nil
}
