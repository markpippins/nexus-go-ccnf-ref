package main

type Extractor interface {
	Extract() ([]Node, error)
	Name() string
	Version() string
}
