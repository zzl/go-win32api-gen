package gomodel

import "go-win32api-gen/utils"

type TypeKind int

const (
	TypeKindOther   TypeKind = 0
	TypeKindPointer TypeKind = 1
	TypeKindIntPtr  TypeKind = 2
	TypeKindStruct  TypeKind = 3
	TypeKindFunc    TypeKind = 4
)

type TypeInfo struct {
	Name string
	Kind TypeKind
	Size utils.SizeInfo
}

func NewTypeInfo(name string) TypeInfo {
	return TypeInfo{Name: name}
}

func NewPointerTypeInfo(name string) TypeInfo {
	return TypeInfo{Name: name, Kind: TypeKindPointer}
}

func (me TypeInfo) String() string {
	return me.Name
}

func (me TypeInfo) IsPointer() bool {
	return me.Kind == TypeKindPointer
}

func (me TypeInfo) IsIntPtr() bool {
	return me.Kind == TypeKindIntPtr
}

func (me TypeInfo) IsStruct() bool {
	return me.Kind == TypeKindStruct
}

func (me TypeInfo) IsFunc() bool {
	return me.Kind == TypeKindFunc
}

type Alias struct {
	Name     string
	RealName string
}

type EnumValue struct {
	Name  string
	Value string
}

type Enum struct {
	Name     string
	BaseType string
	Flags    bool
	Values   []EnumValue
}

type StructField struct {
	Name string
	Type TypeInfo
}

type UnionField struct {
	Name string
	Type TypeInfo
}

type Struct struct {
	Name   string
	Fields []StructField

	UnionFields []UnionField
}

type Com struct {
	Name string
	IID  string

	Super   string
	Methods []Func
}
