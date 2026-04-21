package cmd

import (
	"encoding/json"
	"os"
)

type SubscriptionSet struct {
	Groups  []string `json:"groups"`
	Courses []string `json:"courses"`
}

type SetsConfig map[string]SubscriptionSet

func loadSetsConfig(path string) (SetsConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(SetsConfig), nil
		}
		return nil, err
	}

	var sets SetsConfig
	if err := json.Unmarshal(data, &sets); err != nil {
		return nil, err
	}
	return sets, nil
}
