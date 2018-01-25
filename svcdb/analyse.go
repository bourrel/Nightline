package svcdb

// Consommation
type dataPointC struct {
	
}

type AnalyseC struct {
	ID		int64		`json:"id"`
	Soiree	int64		`json:"soiree"`
	Type	string		`json:"type"`
	
}

// Frequentation

// Population
type AnalyseP struct {
	ID		int64		`json:"id"`
	Soiree	int64		`json:"soiree"`
	Type	string		`json:"type"`
	Values	map[string]int64	`json:"values"`
}
