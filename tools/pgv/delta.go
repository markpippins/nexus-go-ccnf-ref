package main

type IRDelta struct {
	SchemaVersion   string
	IRSchemaVersion string
	BaseHash        string
	HeadHash        string
	Status          string
	Summary         Summary
	Added           []IRNode
	Removed         []IRNode
	Moved           []Move
	Unchanged       []IRNode
}

type Summary struct {
	AddedCount     int
	RemovedCount   int
	MovedCount     int
	UnchangedCount int
}

type Move struct {
	NodeID   string
	FromPath string
	ToPath   string
}
