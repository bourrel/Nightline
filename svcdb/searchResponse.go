package svcdb

// SearchResponse model
type SearchResponse struct {
	ID      int64  `json:"id"`
	Name    string `json:"name"`
	Picture string `json:"picture,omitempty"`
}

// SearchResponses SearchResponse array
type SearchResponses []SearchResponse

func (sr *SearchResponse) FromEstablishment(e Establishment) {
	sr.ID = e.ID
	sr.Name = e.Name
	sr.Picture = e.Image
}

func (sr *SearchResponse) FromUser(u User) {
	sr.ID = u.ID
	sr.Name = u.Pseudo
	sr.Picture = u.Image
}
