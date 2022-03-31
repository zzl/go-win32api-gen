package jsonmodel

type Api struct {
	Name string

	Constants      []*Constant
	Types          []*Type
	Functions      []*Function
	UnicodeAliases []string
}
