package gomodel

type Param struct {
	Name  string
	Type  TypeInfo
	Attrs string
}

type Func struct {
	Name        string
	Params      []Param
	ReturnType  TypeInfo
	ReturnError bool
	Dll         string
}
