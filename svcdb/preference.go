package svcdb

import (
	"encoding/json"

	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
)

// Preference model
type Preference struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// Preferences Preference array
type Preferences []Preference

func (s *Preference) NodeToPreference(node graph.Node) {
	s.ID = node.NodeIdentity
	s.Name = node.Properties["Name"].(string)
}

func (s Preference) MarshalJSON() ([]byte, error) {
	type Alias Preference
	cstruct, err := json.Marshal(&struct {
		Alias
	}{
		Alias: (Alias)(s),
	})
	return cstruct, err
}

func (s *Preference) UnmarshalJSON(data []byte) error {
	type Alias Preference
	aux := &struct {
		// Begin string `json:"begin"`
		// End   string `json:"end"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	return nil
}
