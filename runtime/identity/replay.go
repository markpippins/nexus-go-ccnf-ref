package identity

import "fmt"

func Replay(store Store, events []string) error {
	for i, event := range events {
		if event == "" {
			return fmt.Errorf("replay: empty event at index %d", i)
		}
	}
	return nil
}
