package main

import (
	"bytes"
	"fmt"
	"go-win32api-gen/codegen"
	"go-win32api-gen/gomodel"
	"go-win32api-gen/jsonmodel"
	"go-win32api-gen/utils"
	"io/ioutil"
	"os"
	"strconv"
	"strings"
)

var gTypeInfoMap map[string]*jsonmodel.Type

func MapGoTypeInfo(t *jsonmodel.Type) gomodel.TypeInfo {
	gti := _mapGoTypeInfo(t)
	gTypeInfoMap[gti.Name] = t
	return gti
}

func _mapGoTypeInfo(t *jsonmodel.Type) gomodel.TypeInfo {
	switch t.Kind {
	case "ApiRef":
		//special
		if t.Name == "LARGE_INTEGER" {
			return gomodel.NewTypeInfo("int64")
		}
		if t.Name == "ULARGE_INTEGER" {
			return gomodel.NewTypeInfo("uint64")
		}
		if t.TargetKind == "Com" {
			return gomodel.NewPointerTypeInfo("*" + t.Name)
		}
		if t.ContextType != nil {
			fqRefName := t.ContextType.FqName + "." + t.Name
			if t1, ok := jsonmodel.TypeRegistry[fqRefName]; ok {
				name := ""
				parts := strings.Split(fqRefName[len(t.Api)+1:], ".")
				for n, p := range parts {
					if n > 0 {
						name += "_"
					}
					name += utils.CapName(p)
				}
				gti := _mapGoTypeInfo(t1)
				gti.Name = name
				return gti
			} else {
				//?panic("?")
			}
		}
		name := utils.CapName(t.Name)
		gti := gomodel.NewTypeInfo(name)
		if t.IsPointer() {
			gti.Kind = gomodel.TypeKindPointer
		} else if t.IsIntPointer() {
			gti.Kind = gomodel.TypeKindIntPtr
		} else if t.IsFunctionPointer() {
			gti.Kind = gomodel.TypeKindFunc
		} else if t.IsStruct() {
			gti.Kind = gomodel.TypeKindStruct
			tSize, aSize := t.GetSize()
			gti.Size = utils.SizeInfo{tSize, aSize}
		}
		return gti
	case "Native":
		goType := jsonmodel.MapNativeGoType(t.Name)
		gti := gomodel.NewTypeInfo(goType)
		if goType == "syscall.GUID" {
			gti.Kind = gomodel.TypeKindStruct
			gti.Size = utils.SizeInfo{16, 4}
		}
		return gti
	case "PointerTo", "LPArray":
		if t.Child.Name == "Void" {
			return gomodel.NewPointerTypeInfo("unsafe.Pointer")
		}
		toTypeInfo := MapGoTypeInfo(t.Child)
		if toTypeInfo.Name == "unsafe.Pointer" {
			return toTypeInfo
		}
		return gomodel.NewPointerTypeInfo("*" + toTypeInfo.Name)
	case "Array":
		count := t.Shape.Size
		if count == 0 {
			count = 1
		}
		goType := "[" + strconv.Itoa(count) + "]" + MapGoTypeInfo(t.Child).Name
		return gomodel.NewPointerTypeInfo(goType)
	case "Void":
		return gomodel.NewTypeInfo("")
	case "Struct", "Union":
		gti := gomodel.NewTypeInfo(t.Name)
		gti.Kind = gomodel.TypeKindStruct
		tSize, aSize := t.GetSize()
		gti.Size = utils.SizeInfo{tSize, aSize}
		return gti
	default:
		panic("unknown kind " + t.Kind)
	}
}

func buildConstValue(c *jsonmodel.Constant) string {
	if c.Type.Name == "Guid" {
		return utils.BuildGuidExpr(c.Value.Str)
	}
	//?
	if c.Type.Name == "DEVPROPKEY" {
		return ""
	}
	sValue := c.Value.String()

	if c.Type.IsUnsigned() {
		gti := MapGoTypeInfo(c.Type)
		goType := jsonmodel.MapNativeGoType(c.ValueType)

		if strings.HasPrefix(goType, "int") {
			bitSize, _ := strconv.Atoi(goType[3:])
			nValue, _ := strconv.ParseInt(sValue, 10, bitSize)
			if nValue < 0 {
				sValue = fmt.Sprintf("%v", -nValue-1)
				sValue = "^" + gti.Name + "(" + sValue + ")"
			}
		} else {
			//panic("??")
		}
	}
	return sValue
}

func buildGoApi(api *jsonmodel.Api) *gomodel.GoApi {
	goApi := &gomodel.GoApi{}
	goApi.Name = api.Name

	typeNameSet := make(map[string]bool)

	for _, it := range api.Constants {
		//special..
		if it.Type.Name == "PROPERTYKEY" {
			continue
		}
		sValue := buildConstValue(it)

		cti := MapGoTypeInfo(it.Type)
		c := gomodel.Const{
			Name:  utils.CapName(it.Name),
			Value: sValue,
			Type:  cti.Name,
		}
		if it.Type.IsPointer() || cti.IsStruct() {
			goApi.VarConsts = append(goApi.VarConsts, c)
		} else {
			goApi.Consts = append(goApi.Consts, c)
		}
	}

	structNameMap := make(map[string]bool)
	for _, t := range api.Types {
		goTypeName := utils.CapName(t.Name)
		switch t.Kind {
		case "NativeTypedef":
			typeAlias := gomodel.Alias{
				Name:     goTypeName,
				RealName: MapGoTypeInfo(t.Def).Name,
			}
			goApi.TypeAliases = append(goApi.TypeAliases, typeAlias)
		case "Enum":
			enum := gomodel.Enum{
				Name:     goTypeName,
				Flags:    t.Flags,
				BaseType: jsonmodel.MapNativeGoType(t.IntegerBase),
			}
			for _, v := range t.Values {
				value := gomodel.EnumValue{
					Name:  utils.CapName(v.Name),
					Value: v.Value.String(),
				}
				enum.Values = append(enum.Values, value)
			}
			goApi.Enums = append(goApi.Enums, enum)
		case "Struct":
			ss := buildGoStruct(t, "", typeNameSet)
			goApi.Structs = append(goApi.Structs, ss...)
			for _, s := range ss {
				structNameMap[s.Name] = true
			}
		case "Union":
			ss := buildUnionStructs(t, "", typeNameSet)
			goApi.Structs = append(goApi.Structs, ss...)
		case "Com":
			c := buildCom(t, typeNameSet)
			goApi.Coms = append(goApi.Coms, c)
		case "ComClassID":
			//
		case "FunctionPointer":
			gf := gomodel.Func{
				Name: t.Name,
			}
			for _, p := range t.Params {
				gp := gomodel.Param{
					Name: utils.SafeGoName(p.Name),
					Type: MapGoTypeInfo(p.Type),
				}
				gf.Params = append(gf.Params, gp)
			}
			if t.ReturnType != nil {
				gf.ReturnType = MapGoTypeInfo(t.ReturnType)
			}
			goApi.FuncTypes = append(goApi.FuncTypes, gf)
		default:
			panic("?")
		}
	}

	funcNameMap := make(map[string]bool)
	for _, it := range api.Functions {
		gf := gomodel.Func{
			Name: it.Name,
		}
		for _, p := range it.Params {
			ti := MapGoTypeInfo(p.Type)
			gp := gomodel.Param{
				Name:  utils.SafeGoName(p.Name),
				Type:  ti,
				Attrs: jsonmodel.BuildAttrsStr(p.Attrs),
			}
			typeNameSet[ti.Name] = true
			gf.Params = append(gf.Params, gp)
		}
		if it.ReturnType != nil {
			ti := MapGoTypeInfo(it.ReturnType)
			gf.ReturnType = ti
			typeNameSet[ti.Name] = true
		}
		if it.SetLastError {
			gf.ReturnError = true
		}
		gf.Dll = it.DllImport
		funcNameMap[gf.Name] = true
		goApi.Funcs = append(goApi.Funcs, gf)
	}

	for _, a := range api.UnicodeAliases {
		a = utils.CapName(a)
		uName := a + "W"
		if _, ok := structNameMap[uName]; ok {
			goApi.StructAliases = append(goApi.StructAliases, gomodel.Alias{
				Name:     a,
				RealName: uName,
			})
		} else if _, ok := funcNameMap[uName]; ok {
			goApi.FuncAliases = append(goApi.FuncAliases, gomodel.Alias{
				Name:     a,
				RealName: uName,
			})
		}
	}

	//
	hasSyscall := false
	hasUnsafe := false
	if len(goApi.Funcs) > 0 || len(goApi.Coms) > 0 {
		hasSyscall = true
	}
	delete(typeNameSet, "")
	for k, _ := range typeNameSet {
		if strings.Contains(k, "unsafe.") || k[0] == '*' {
			hasUnsafe = true
		} else if strings.Contains(k, "syscall.") {
			hasSyscall = true
		}
	}
	for _, s := range goApi.Structs {
		if len(s.UnionFields) > 0 {
			hasUnsafe = true
		}
	}
	if hasUnsafe {
		goApi.Imports = append(goApi.Imports, "unsafe")
	}
	if hasSyscall {
		goApi.Imports = append(goApi.Imports, "syscall")
	}

	return goApi
}

func buildCom(t *jsonmodel.Type, set map[string]bool) gomodel.Com {
	com := gomodel.Com{
		Name: t.Name,
	}
	com.IID = t.Guid
	if t.Interface != nil {
		com.Super = t.Interface.Name
	}
	for _, method := range t.Methods {
		gm := gomodel.Func{
			Name: method.Name,
		}
		for _, p := range method.Params {
			gp := gomodel.Param{
				Name: utils.SafeGoName(p.Name),
				Type: MapGoTypeInfo(p.Type),
			}
			gm.Params = append(gm.Params, gp)
		}
		if method.ReturnType != nil {
			gm.ReturnType = MapGoTypeInfo(method.ReturnType)
		}
		com.Methods = append(com.Methods, gm)
	}
	return com
}

func buildUnionStructs(t *jsonmodel.Type, parentGoTypeName string,
	typeNameSet map[string]bool) []gomodel.Struct {

	goTypeName := getGoTypeName(parentGoTypeName, t)

	var ss []gomodel.Struct

	ss = buildNestedTypes(goTypeName, t, typeNameSet)

	s := gomodel.Struct{
		Name: goTypeName,
	}

	size, alignSize := t.GetSize()
	if alignSize == 0 {
		if size > 8 {
			panic("?")
		}
		alignSize = size
	}
	embedFieldIndex := -1
	for n, f := range t.Fields {
		if f.Name == "Anonymous" {
			fSize, fAlign := f.Type.GetSize()
			if fSize == size {
				embedFieldIndex = n
			} else {
				_ = fAlign
				//?
			}
			break
		}
	}
	if embedFieldIndex != -1 {
		f := t.Fields[embedFieldIndex]
		s.Fields = append(s.Fields, gomodel.StructField{
			Name: "",
			Type: MapGoTypeInfo(f.Type),
		})
	} else {
		var elemType string
		switch alignSize {
		case 1:
			elemType = "byte"
		case 2:
			elemType = "uint16"
		case 4:
			elemType = "uint32"
		case 8:
			elemType = "uint64"
		default:
			panic("?")
		}
		elemCount := size / alignSize
		goType := fmt.Sprintf("[%d]%s", elemCount, elemType)
		ti := gomodel.NewPointerTypeInfo(goType)
		s.Fields = append(s.Fields, gomodel.StructField{
			Name: "Data",
			Type: ti,
		})
	}

	for n, f := range t.Fields {
		if n == embedFieldIndex {
			continue
		}
		s.UnionFields = append(s.UnionFields, gomodel.UnionField{
			Name: utils.CapName(f.Name),
			Type: MapGoTypeInfo(f.Type),
		})
	}

	ss = append(ss, s)
	return ss
}

func getGoTypeName(parentGoTypeName string, t *jsonmodel.Type) string {
	goTypeName := utils.CapName(t.Name)
	if parentGoTypeName != "" {
		goTypeName = parentGoTypeName + "_" + goTypeName
	}
	return goTypeName
}

func buildNestedTypes(parentGoTypeName string,
	parentType *jsonmodel.Type, typeNameSet map[string]bool) []gomodel.Struct {
	var ss []gomodel.Struct
	for _, nestedType := range parentType.NestedTypes {
		if nestedType.Kind == "Struct" {
			nestedSs := buildGoStruct(nestedType, parentGoTypeName, typeNameSet)
			ss = append(ss, nestedSs...)
		} else if nestedType.Kind == "Union" {
			nestedSs := buildUnionStructs(nestedType, parentGoTypeName, typeNameSet)
			ss = append(ss, nestedSs...)
		}
	}
	return ss
}

func buildGoStruct(t *jsonmodel.Type, parentGoTypeName string,
	typeNameSet map[string]bool) []gomodel.Struct {

	goTypeName := getGoTypeName(parentGoTypeName, t)

	s := gomodel.Struct{
		Name: goTypeName,
	}

	var ss []gomodel.Struct
	ss = buildNestedTypes(goTypeName, t, typeNameSet)
	for _, it := range t.Fields {
		ti := MapGoTypeInfo(it.Type)
		f := gomodel.StructField{
			Name: utils.CapName(it.Name),
			Type: ti,
		}
		typeNameSet[ti.Name] = true
		s.Fields = append(s.Fields, f)
	}
	ss = append(ss, s)
	return ss
}

func main() {

	gTypeInfoMap = make(map[string]*jsonmodel.Type)

	apis := jsonmodel.LoadApis("win32json/api")

	os.Mkdir("win32", os.ModePerm)
	for _, api := range apis {
		goApi := buildGoApi(api)

		w := bytes.NewBuffer(nil)
		codegen.Gen(goApi, w)
		s := w.String()

		filePath := "win32/" + goApi.Name + ".go"
		ioutil.WriteFile(filePath, []byte(s), os.ModePerm)
	}

	println("Done.")
}
