package gomodel

type GoApi struct {
	Name string

	Imports []string

	TypeAliases []Alias

	Consts []Const

	VarConsts []Const

	Enums []Enum

	Structs []Struct

	FuncTypes []Func

	Funcs []Func

	StructAliases []Alias

	FuncAliases []Alias

	Coms []Com

	//ComClassID?
}
