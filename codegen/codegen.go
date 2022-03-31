package codegen

import (
	"fmt"
	"go-win32api-gen/gomodel"
	"go-win32api-gen/utils"
	"io"
	"strings"
)

func Gen(api *gomodel.GoApi, w io.Writer) {
	fmt.Fprintln(w, "package win32")
	fmt.Fprintln(w)

	genImports(api, w)
	genTypeAliases(api, w)
	genConsts(api, w)
	genVarConsts(api, w)
	genEnums(api, w)
	genStructs(api, w)
	genFuncTypes(api, w)

	genComs(api, w)
	genFuncs(api, w)
}

func libName(dll string) string {
	return "lib" + utils.CapName(strings.ToLower(dll))
}

func genFunc(f gomodel.Func, w io.Writer) {
	goName := utils.CapName(f.Name)
	fmt.Fprint(w, "func ", goName, "(")
	for m, p := range f.Params {
		if m > 0 {
			fmt.Fprint(w, ", ")
		}
		pType := p.Type.Name
		if p.Type.IsFunc() {
			pType = "uintptr"
		}
		fmt.Fprint(w, p.Name, " ", pType)
	}
	fmt.Fprint(w, ")")

	var retIsPtr bool
	var retIsStruct bool
	retType := f.ReturnType.Name
	hasRet := retType != ""
	if hasRet {
		retIsPtr = f.ReturnType.IsPointer()
		retIsStruct = f.ReturnType.IsStruct()
		if f.ReturnType.IsFunc() {
			retType = "uintptr"
		}
	}

	if hasRet && f.ReturnError {
		fmt.Fprint(w, " (", retType, ", WIN32_ERROR)")
	} else if hasRet {
		fmt.Fprint(w, " ", retType)
	} else if f.ReturnError {
		fmt.Fprint(w, " WIN32_ERROR")
	}

	fmt.Fprintln(w, " {")

	fmt.Fprint(w, "\t", "addr := lazyAddr(&p", goName,
		", ", libName(f.Dll), ", \"", f.Name, "\")\n")

	if hasRet {
		fmt.Fprint(w, "\tret, _, ")
	} else {
		fmt.Fprint(w, "\t_, _, ")
	}
	if f.ReturnError {
		fmt.Fprint(w, " err")
	} else {
		fmt.Fprint(w, " _")
	}
	fmt.Fprint(w, " ")
	if hasRet || f.ReturnError {
		fmt.Fprint(w, ":")
	}
	fmt.Fprint(w, "= syscall.SyscallN(addr")
	for _, p := range f.Params {
		fmt.Fprint(w, ", ")
		pType := p.Type.Name
		if p.Type.IsFunc() { //} gomodel.IsFunctionPointer(pType) {
			pType = "uintptr"
		}
		pName := utils.SafeGoName(p.Name)

		if p.Type.IsStruct() {
			size := p.Type.Size.TotalSize
			if size > utils.PtrSize {
				fmt.Fprint(w, "(uintptr)(unsafe.Pointer(&", pName, "))")
			} else {
				fmt.Fprint(w, "*(*uintptr)(unsafe.Pointer(&", pName, "))")
			}
			continue
		}

		if p.Type.IsIntPtr() {
			fmt.Fprint(w, pName)
		} else {
			fmt.Fprint(w, "uintptr(")
			if p.Type.IsPointer() {
				//?
				if pType == "unsafe.Pointer" {
					fmt.Fprint(w, pName)
				} else {
					fmt.Fprint(w, "unsafe.Pointer("+pName+")")
				}
			} else {
				fmt.Fprint(w, pName)
			}
			fmt.Fprint(w, ")")
		}
	}
	fmt.Fprintln(w, ")")
	if hasRet {
		if retType == "uintptr" {
			fmt.Fprint(w, "\treturn ret")
		} else if retIsPtr {
			fmt.Fprint(w, "\treturn (", retType, ")(unsafe.Pointer(ret))")
		} else if retIsStruct {
			fmt.Fprint(w, "\treturn *(*", retType, ")(unsafe.Pointer(ret))")
		} else {
			fmt.Fprint(w, "\treturn ", retType, "(ret)")
		}
		if f.ReturnError {
			fmt.Fprint(w, ", WIN32_ERROR(err)")
		}
		fmt.Fprintln(w, "")
	} else {
		if f.ReturnError {
			fmt.Fprint(w, "\treturn WIN32_ERROR(err)\n")
		}
	}
	fmt.Fprintln(w, "}")
	fmt.Fprintln(w, "")
}

func genFuncs(api *gomodel.GoApi, w io.Writer) {
	fmt.Fprintln(w, "var (")

	aliasMap := make(map[string]string)
	for _, a := range api.FuncAliases {
		aliasMap[a.RealName] = a.Name
	}

	for _, it := range api.Funcs {
		fmt.Fprintln(w, "\tp"+utils.CapName(it.Name), "uintptr")
	}
	fmt.Fprintln(w, ")")
	fmt.Fprintln(w, "")

	for _, f := range api.Funcs {
		if a, ok := aliasMap[f.Name]; ok {
			fmt.Fprintln(w, "var", a, "=", f.Name)
		}
		genFunc(f, w)
	}
	fmt.Fprintln(w)
}

func genComs(api *gomodel.GoApi, w io.Writer) {
	if len(api.Coms) == 0 {
		return
	}
	fmt.Fprintln(w, "// coms")
	fmt.Fprintln(w)
	for _, it := range api.Coms {
		genCom(it, w)
	}
	fmt.Fprintln(w)
}

func genCom(c gomodel.Com, w io.Writer) {
	if c.IID != "" {
		fmt.Fprintln(w, "// "+c.IID)
		expr := utils.BuildGuidExpr(c.IID)
		fmt.Fprint(w, "var IID_", c.Name, " = ", expr, "\n")
		fmt.Fprintln(w)
	}
	fmt.Fprint(w, "type ", c.Name+"Interface", " interface {\n")
	if c.Super != "" {
		fmt.Fprint(w, "\t", c.Super, "Interface\n")
	}
	for _, method := range c.Methods {
		fmt.Fprint(w, "\t", method.Name, "(")
		for m, p := range method.Params {
			if m > 0 {
				fmt.Fprint(w, ", ")
			}
			fmt.Fprint(w, p.Name, " ", p.Type)
		}
		fmt.Fprint(w, ")")

		retType := method.ReturnType.Name
		if retType != "" {
			fmt.Fprint(w, " ", retType)
		}
		fmt.Fprintln(w)
	}
	fmt.Fprint(w, "}\n")
	fmt.Fprintln(w)

	fmt.Fprintln(w, "type", c.Name+"Vtbl", "struct {")
	if c.Super != "" {
		fmt.Fprint(w, "\t", c.Super, "Vtbl\n")
	}
	for _, method := range c.Methods {
		fmt.Fprint(w, "\t", method.Name, " uintptr\n")
	}
	fmt.Fprintln(w, "}")
	fmt.Fprintln(w)

	fmt.Fprint(w, "type ", c.Name, " struct {\n")
	if c.Super == "" {
		if c.Name != "IUnknown" {
			panic("?")
		}
		fmt.Fprintln(w, "\tLpVtbl *[1024]uintptr")
	} else {
		fmt.Fprint(w, "\t", c.Super, "\n")
	}
	fmt.Fprintln(w, "}")
	fmt.Fprintln(w)

	//
	fmt.Fprint(w, "func (this *", c.Name, ") Vtbl() *", c.Name, "Vtbl {\n")

	if c.Super == "" {
		fmt.Fprint(w, "\treturn (*IUnknownVtbl)(unsafe.Pointer(this.LpVtbl))\n")
	} else {
		fmt.Fprint(w, "\t", "return (*", c.Name,
			"Vtbl)(unsafe.Pointer(this.IUnknown.LpVtbl))\n")
	}
	fmt.Fprint(w, "}\n")
	fmt.Fprintln(w)

	//
	for _, method := range c.Methods {
		fmt.Fprint(w, "func (this *", c.Name, ") ", method.Name, "(")
		for m, p := range method.Params {
			if m > 0 {
				fmt.Fprint(w, ", ")
			}

			pType := p.Type.Name
			if p.Type.IsFunc() {
				pType = "uintptr"
			}
			fmt.Fprint(w, p.Name, " ", pType)
		}
		fmt.Fprint(w, ")")

		var hasRet bool
		var retType string
		var retIsPtr bool
		var retIsStruct bool

		retType = method.ReturnType.Name
		if retType != "" {
			fmt.Fprint(w, " ", retType)
			hasRet = true

			retIsPtr = method.ReturnType.IsPointer()
			retIsStruct = method.ReturnType.IsStruct()
		}
		fmt.Fprintln(w, "{")

		if hasRet {
			fmt.Fprint(w, "\tret, _, _ :")

		} else {
			fmt.Fprint(w, "\t_, _, _ ")
		}
		fmt.Fprint(w, "= syscall.SyscallN(this.Vtbl().",
			method.Name, ", uintptr(unsafe.Pointer(this))")

		for _, p := range method.Params {
			fmt.Fprint(w, ", ")
			pType := p.Type.Name
			if p.Type.IsStruct() {
				size := p.Type.Size.TotalSize
				if size > utils.PtrSize {
					fmt.Fprint(w, "(uintptr)(unsafe.Pointer(&", p.Name, "))")
				} else {
					fmt.Fprint(w, "*(*uintptr)(unsafe.Pointer(&", p.Name, "))")
				}
				continue
			}
			if p.Type.IsFunc() {
				pType = "uintptr"
			}

			if pType == "uintptr" {
				fmt.Fprint(w, p.Name)
			} else {
				fmt.Fprint(w, "uintptr(")
				if p.Type.IsPointer() {
					fmt.Fprint(w, "unsafe.Pointer("+p.Name+")")
				} else {
					fmt.Fprint(w, p.Name)
				}
				fmt.Fprint(w, ")")
			}
		}
		fmt.Fprintln(w, ")")
		if hasRet {
			if retIsPtr {
				fmt.Fprint(w, "\treturn (", retType, ")(")
				if retType != "unsafe.Pointer" {
					fmt.Fprint(w, "unsafe.Pointer(ret)")
				} else {
					fmt.Fprint(w, "ret")
				}
				fmt.Fprint(w, ")")
			} else if retIsStruct {
				fmt.Fprint(w, "\treturn *(*", retType, ")(unsafe.Pointer(ret))")
			} else if retType == "bool" {
				fmt.Fprint(w, "\treturn ret != 0")
			} else {
				fmt.Fprint(w, "\treturn ", retType, "(ret)")
			}
		}
		fmt.Fprintln(w, "")
		fmt.Fprintln(w, "}")
		fmt.Fprintln(w, "")
	}
}

func genFuncTypes(api *gomodel.GoApi, w io.Writer) {
	if len(api.FuncTypes) == 0 {
		return
	}
	fmt.Fprintln(w, "// func types")
	fmt.Fprintln(w)
	for _, it := range api.FuncTypes {
		fmt.Fprint(w, "type ", it.Name, " func(")
		for m, p := range it.Params {
			if m > 0 {
				fmt.Fprint(w, ", ")
			}
			fmt.Fprint(w, p.Name, " ", p.Type)
		}
		fmt.Fprint(w, ")")
		retType := it.ReturnType.Name
		if retType != "" {
			fmt.Fprint(w, " ", retType)
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w)
}

func genStructs(api *gomodel.GoApi, w io.Writer) {
	if len(api.Structs) == 0 {
		return
	}
	aliasMap := make(map[string]string)
	for _, a := range api.StructAliases {
		aliasMap[a.RealName] = a.Name
	}
	fmt.Fprintln(w, "// structs")
	fmt.Fprintln(w)
	for _, it := range api.Structs {
		if a, ok := aliasMap[it.Name]; ok {
			fmt.Fprintln(w, "type", a, "=", it.Name)
		}

		fmt.Fprintln(w, "type", it.Name, "struct {")
		for _, f := range it.Fields {
			fType := f.Type.Name
			if f.Type.IsFunc() {
				fType = "uintptr"
			}
			if strings.HasPrefix(f.Name, "Anonymous") {
				fmt.Fprintln(w, "\t"+fType)
			} else {
				fmt.Fprintln(w, "\t"+f.Name, fType)
			}
		}
		fmt.Fprintln(w, "}")
		fmt.Fprintln(w)

		for _, uf := range it.UnionFields {
			fmt.Fprint(w, "func (this *", it.Name, ") ",
				uf.Name, "() *", uf.Type, "{\n")
			fmt.Fprint(w, "\treturn (*", uf.Type, ")(unsafe.Pointer(this))\n")
			fmt.Fprintln(w, "}")
			fmt.Fprintln(w)

			fmt.Fprint(w, "func (this *", it.Name, ") ",
				uf.Name, "Val() ", uf.Type, "{\n")
			fmt.Fprint(w, "\treturn *(*", uf.Type, ")(unsafe.Pointer(this))\n")
			fmt.Fprintln(w, "}")
			fmt.Fprintln(w)
		}
	}
	fmt.Fprintln(w)
}

func genEnums(api *gomodel.GoApi, w io.Writer) {
	if len(api.Enums) == 0 {
		return
	}
	fmt.Fprintln(w, "// enums")
	fmt.Fprintln(w)
	for _, it := range api.Enums {

		fmt.Fprintln(w, "// enum", it.Name)
		if it.Flags {
			fmt.Fprintln(w, "// flags")
		}

		typeName := it.Name
		fmt.Fprintln(w, "type", typeName, it.BaseType)
		fmt.Fprintln(w, "const (")
		for _, v := range it.Values {
			sValue := v.Value
			vName := v.Name
			fmt.Fprintln(w, "\t"+vName, typeName, "=", sValue)
		}
		fmt.Fprintln(w, ")")
		fmt.Fprintln(w)
	}
	fmt.Fprintln(w)
}

func genVarConsts(api *gomodel.GoApi, w io.Writer) {
	if len(api.VarConsts) == 0 {
		return
	}
	fmt.Fprintln(w, "var (")
	for _, it := range api.VarConsts {
		sValue := it.Value
		if it.Type != "syscall.GUID" {
			sValue = it.Type + "(" + "unsafe.Pointer(uintptr(" + sValue + ")))"
		}
		fmt.Fprintln(w, "\t"+it.Name, it.Type, "=", sValue)
	}
	fmt.Fprintln(w, ")")
	fmt.Fprintln(w)
}

func genTypeAliases(api *gomodel.GoApi, w io.Writer) {
	if len(api.TypeAliases) == 0 {
		return
	}
	for _, it := range api.TypeAliases {
		fmt.Fprintln(w, "type", it.Name, "=", it.RealName)
	}
	fmt.Fprintln(w)
}

func genImports(api *gomodel.GoApi, w io.Writer) {
	if len(api.Imports) == 0 {
		return
	}
	for _, it := range api.Imports {
		fmt.Fprintln(w, "import \""+it+"\"")
	}
	fmt.Fprintln(w)
}

func genConsts(api *gomodel.GoApi, w io.Writer) {
	if len(api.Consts) == 0 {
		return
	}
	fmt.Fprintln(w, "const (")
	for _, it := range api.Consts {
		fmt.Fprintln(w, "\t"+it.Name, it.Type, "=", it.Value)
	}
	fmt.Fprintln(w, ")")
	fmt.Fprintln(w)
}
