package view

type View interface {
	Kind() string
}

type ExampleView struct {
	Key   string
	Value string
}

func (ExampleView) Kind() string { return "example" }
