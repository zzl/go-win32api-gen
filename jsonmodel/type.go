package jsonmodel

import (
	"go-win32api-gen/utils"
	"log"
	"math/big"
	"unsafe"
)

var TypeRegistry map[string]*Type

const PtrSize = utils.PtrSize

type Type struct {
	Name          string
	FqName        string
	Architectures []string
	Platform      string
	Kind          string
	Scoped        bool

	NullNullTerm    bool
	CountConst      int
	CountParamIndex int
	Parents         []string
	Child           *Type
	NestedTypes     []*Type

	//
	ContextType *Type
	//RefType     *Type
	_refType *Type

	//ApiRef
	TargetKind string
	Api        string

	//
	Parent *Type

	//Array
	Shape struct {
		Size int
	}

	//NativeTypedef
	Def           *Type
	AlsoUsableFor string
	FreeFunc      string

	//Enum
	Flags       bool
	IntegerBase string
	Values      []struct {
		Name  string
		Value big.Int
	}

	//Struct
	//Size int
	PackingSize int
	Fields      []*struct {
		Name  string
		Type  *Type
		Attrs []Attr
	}

	//Com
	Guid      string
	Interface *Type
	Methods   []*Function

	//FunctionPointer
	ReturnType   *Type
	SetLastError bool
	Params       []*struct {
		Name  string
		Type  *Type
		Attrs []Attr
	}
}

func (this *Type) String() string {
	if this.Child != nil {
		if this.Name != "" {
			log.Fatal("?")
		}
		return "?" + this.Child.Name
	} else {
		return this.Name
	}
}

//
func (t *Type) IsPointer() bool {
	switch t.Kind {
	case "ApiRef":
		refType := t.GetRefType()
		if refType == nil { //?
			return false
		}
		return refType.IsPointer()
	case "NativeTypedef":
		return t.Def.IsPointer()
	case "PointerTo":
		return true
	case "LPArray":
		return true
	case "Array":
		return true
	default:
		return false
	}
}

func (this *Type) IsFunctionPointer() bool {
	if this.Kind == "FunctionPointer" {
		return true
	} else if this.Kind == "ApiRef" && this.TargetKind != "Com" {
		//special
		if this.Name == "HKL" || this.Name == "HTASK" {
			return false
		}
		refType := this.GetRefType()
		if refType == nil {
			return false
		}
		return refType.IsFunctionPointer()
	}
	return false
}

//
func (this *Type) IsIntPointer() bool {
	if this.Kind == "Native" && (this.Name == "IntPtr" || this.Name == "UIntPtr") {
		return true
	} else if this.Kind == "NativeTypedef" {
		return this.Def.IsIntPointer()
	} else if this.Kind == "ApiRef" {
		//special
		if this.Name == "HKL" {
			return true
		}
		refType := this.GetRefType()
		if refType == nil {
			return false
		}
		return refType.IsIntPointer()
	}
	return false
}

func (this *Type) IsUnsigned() bool {
	if this.Kind == "Native" {
		if this.Name == "IntPtr" || this.Name == "UIntPtr" {
			return true
		}
		return this.Name == "Byte" || this.Name == "UInt16" ||
			this.Name == "Char" || this.Name == "UInt32" ||
			this.Name == "UInt64" ||
			this.Name == "UIntPtr" || this.Name == "IntPtr"
	} else if this.Kind == "NativeTypedef" {
		return this.Def.IsUnsigned()
	} else if this.Kind == "ApiRef" {
		//special
		if this.Name == "HKL" || this.Name == "DEVPROPKEY" {
			return true
		}
		return this.GetRefType().IsUnsigned()
	}
	return false
}

func (this *Type) IsStruct() bool {
	if this.Kind == "ApiRef" && this.TargetKind != "Com" {
		if this.Name == "HKL" {
			return false
		}
		refType := this.GetRefType()
		if refType == nil {
			return false
		}
		return refType.IsStruct()
	} else if this.Kind == "Struct" {
		return true
	} else if this.Kind == "Union" { //..
		return true
	} else if this.Kind == "Native" && this.Name == "Guid" {
		return true
	}
	return false
}

//
func (this *Type) GetRefType() *Type {
	if this._refType != nil {
		return this._refType
	}
	if this.ContextType != nil {
		fqRefName := this.ContextType.FqName + "." + this.Name
		if refType, ok := TypeRegistry[fqRefName]; ok {
			this._refType = refType
			return this._refType
		}
	}
	refFqName := this.Api + "."
	for _, p := range this.Parents {
		refFqName += p + "."
	}
	refFqName += this.Name
	this._refType = TypeRegistry[refFqName]
	return this._refType
}

func getSizeOfNativeType(name string) int {
	switch name {
	case "Char", "Byte", "SByte", "Boolean":
		return 1
	case "Int16", "UInt16":
		return 2
	case "Int32", "UInt32", "Single":
		return 4
	case "Int64", "UInt64", "IntPtr":
		return 8
	case "UIntPtr", "Double":
		return int(unsafe.Sizeof(uintptr(0)))
	case "Guid":
		return 16
	default:
		panic("unknown nantive type " + name)
	}
}

func (t *Type) GetSize() (int, int) {
	switch t.Kind {
	case "Native":
		size := getSizeOfNativeType(t.Name)
		if t.Name == "Guid" {
			return size, 4
		}
		return size, size
	case "PointerTo":
		return PtrSize, PtrSize
	case "LPArray":
		return PtrSize, PtrSize //?
	case "Array":
		size, alignSize := t.Child.GetSize()
		count := t.Shape.Size
		if count == 0 {
			count = 1
		}
		return size * count, alignSize
	case "ApiRef":
		if t.ContextType != nil {
			fqRefName := t.ContextType.FqName + "." + t.Name
			if refType, ok := TypeRegistry[fqRefName]; ok {
				return refType.GetSize()
			}
		}
		fqRefName := t.Api + "." + t.Name
		if refType, ok := TypeRegistry[fqRefName]; ok {
			return refType.GetSize()
		}
		//special
		if t.Name == "POINTER_TOUCH_INFO" {
			return 144, 8
		} else if t.Name == "POINTER_PEN_INFO" {
			return 120, 8
		}
		panic("?")
	case "NativeTypedef":
		if t.Def.Kind == "PointerTo" {
			return PtrSize, PtrSize
		}
		size := getSizeOfNativeType(t.Def.Name)
		return size, size
	case "Enum":
		size := getSizeOfNativeType(t.IntegerBase)
		return size, size
	case "Com":
		return PtrSize, PtrSize //?
	case "FunctionPointer":
		return PtrSize, PtrSize //?
	case "Struct":
		var fieldSis []utils.SizeInfo
		for _, f := range t.Fields {
			size, alignSize := f.Type.GetSize()
			fieldSis = append(fieldSis, utils.SizeInfo{size, alignSize})
		}
		ssi := utils.StructSize(fieldSis...)
		return ssi.TotalSize, ssi.AlignSize
	case "Union":
		var maxSize int
		maxAlignSize := 0
		for _, f := range t.Fields {
			size, alignSize := f.Type.GetSize()
			if alignSize == 0 {
				if size > 8 {
					panic("?")
				}
				alignSize = size
			}
			if size > maxSize {
				maxSize = size
			}
			if alignSize > maxAlignSize {
				maxAlignSize = alignSize
			}
		}
		return maxSize, maxAlignSize
	default:
		panic("?")
	}
}
