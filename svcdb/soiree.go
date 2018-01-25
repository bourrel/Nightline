package svcdb

import (
	"encoding/json"
	"time"

	"github.com/johnnadratowski/golang-neo4j-bolt-driver/structures/graph"
)

// Soiree model
type Soiree struct {
	ID      int64     `json:"id"`
	Desc    string    `json:"desc"`
	Begin   time.Time `json:"begin"`
	End     time.Time `json:"end"`
	Menu    Menu      `json:"menu"`
	Friends []Profile `json:"friends,omitempty"`
}

// Soirees Soiree array
type Soirees []Soiree

func (s *Soiree) NodeToSoiree(node graph.Node) {
	s.ID = node.NodeIdentity
	s.Desc = node.Properties["Desc"].(string)
	s.Begin, _ = time.Parse(timeForm, node.Properties["Begin"].(string))
	s.End, _ = time.Parse(timeForm, node.Properties["End"].(string))
}

func (s Soiree) MarshalJSON() ([]byte, error) {
	type Alias Soiree
	cstruct, err := json.Marshal(&struct {
		Alias
		Begin string `json:"begin"`
		End   string `json:"end"`
	}{
		Alias: (Alias)(s),
		Begin: s.Begin.Format(timeForm),
		End:   s.End.Format(timeForm),
	})
	return cstruct, err
}

func (s *Soiree) UnmarshalJSON(data []byte) error {
	type Alias Soiree
	aux := &struct {
		Begin string `json:"begin"`
		End   string `json:"end"`
		*Alias
	}{
		Alias: (*Alias)(s),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	if len(aux.Begin) > 0 {
		begin, err := time.Parse(timeForm, aux.Begin)
		if err != nil {
			begin, err = time.Parse(timeFormFucked, aux.Begin)
			if err != nil {
				return err
			}
		}
		s.Begin = begin
	}
	if len(aux.End) > 0 {
		end, err := time.Parse(timeForm, aux.End)
		if err != nil {
			end, err = time.Parse(timeFormFucked, aux.End)
			if err != nil {
				return err
			}
		}
		s.End = end
	}
	return nil
}
