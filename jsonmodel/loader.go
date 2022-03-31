package jsonmodel

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"strings"
)

func LoadApis(dir string) []*Api {
	var apis []*Api

	fis, _ := ioutil.ReadDir(dir)
	for _, fi := range fis {
		var api Api
		name := fi.Name()
		pos := strings.LastIndexByte(name, '.')
		api.Name = name[:pos]

		filePath := dir + "/" + name
		sJson, err := ioutil.ReadFile(filePath)
		if err != nil {
			log.Fatal(err)
		}
		err = json.Unmarshal(sJson, &api)
		if err != nil {
			log.Fatal(err)
		}
		preprocessApi(&api)
		apis = append(apis, &api)
	}
	TypeRegistry = buildTypeRegistry(apis)
	return apis
}

func ignoreArch(arches []string) bool {
	if len(arches) == 0 {
		return false
	}
	for _, it := range arches {
		if it == "X64" {
			return false
		}
	}
	return true
}

func preprocessTypes(types []*Type) []*Type {
	var newTypes []*Type
	for _, t := range types {
		if ignoreArch(t.Architectures) {
			continue
		}
		t.NestedTypes = preprocessTypes(t.NestedTypes)
		preprocessType(t)
		newTypes = append(newTypes, t)
	}
	return newTypes
}

func setTypeRefContextType(t *Type, contextType *Type) {
	if t == nil {
		return
	}
	t.ContextType = contextType
	setTypeRefContextType(t.Child, contextType)
	setTypeRefContextType(t.Def, contextType)
}

func preprocessType(t *Type) {
	for _, nt := range t.NestedTypes {
		nt.ContextType = t
	}
	for _, f := range t.Fields {
		setTypeRefContextType(f.Type, t)
	}
	for _, m := range t.Methods {
		for _, p := range m.Params {
			setTypeRefContextType(p.Type, t)
		}
		setTypeRefContextType(m.ReturnType, t)
	}
}

func preprocessApi(api *Api) {
	api.Types = preprocessTypes(api.Types)
	api.Functions = preprocessFunctions(api.Functions)
}

func preprocessFunctions(fs []*Function) []*Function {
	dlls := ",advapi32,comctl32,comdlg32,gdi32,msimg32,gdiplus," +
		"kernel32,ole32,oleaut32,pdh,shell32,shlwapi,user32,uxtheme," +
		"version,userenv,"
	var newFs []*Function
	for _, f := range fs {
		if ignoreArch(f.Architectures) {
			continue
		}
		dll := strings.ToLower(f.DllImport)
		if !strings.Contains(dlls, ","+dll+",") {
			continue
		}
		newFs = append(newFs, f)
	}
	return newFs
}

//

func collectTypeMap(path string, parentType *Type,
	types []*Type, typeMap map[string]*Type) {

	for _, t := range types {
		t.Parent = parentType
		key := path + "." + t.Name
		if t0, ok := typeMap[key]; ok {
			println(t0.Name)
			panic("?")
		}
		typeMap[key] = t
		collectTypeMap(key, t, t.NestedTypes, typeMap)
	}
}

//fq name as key
func buildTypeRegistry(apis []*Api) map[string]*Type {
	reg := make(map[string]*Type)

	for _, api := range apis {
		collectTypeMap(api.Name, nil, api.Types, reg)
	}
	for k, t := range reg {
		t.FqName = k
	}
	return reg
}
