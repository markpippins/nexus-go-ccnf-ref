package main

type IRDelta struct {
	SchemaVersion   string   `json:"schema_version"`
	IRSchemaVersion string   `json:"ir_schema_version"`
	BaseHash        string   `json:"base_hash"`
	HeadHash        string   `json:"head_hash"`
	Status          string   `json:"status"`
	Summary         Summary  `json:"summary"`
	Added           []IRNode `json:"added"`
	Removed         []IRNode `json:"removed"`
	Moved           []Move   `json:"moved"`
	Unchanged       []IRNode `json:"unchanged"`
}

type Summary struct {
	AddedCount     int `json:"added_count"`
	RemovedCount   int `json:"removed_count"`
	MovedCount     int `json:"moved_count"`
	UnchangedCount int `json:"unchanged_count"`
}

type Move struct {
	NodeID   string `json:"node_id"`
	FromPath string `json:"from_path"`
	ToPath   string `json:"to_path"`
}
