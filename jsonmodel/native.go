package jsonmodel

func MapNativeGoType(t string) string {
	switch t {
	case "SByte":
		return "int8"
	case "Byte":
		return "uint8"
	case "Int16":
		return "int16"
	case "UInt16":
		return "uint16"
	case "Char":
		return "uint16"
	case "Int32":
		return "int32"
	case "UInt32":
		return "uint32"
	case "Int64":
		return "int64"
	case "UInt64":
		return "uint64"
	case "Single":
		return "float32"
	case "Double":
		return "float64"
	case "IntPtr":
		return "uintptr" //?
	case "UIntPtr":
		return "uintptr"
	case "Guid":
		return "syscall.GUID"
	case "Void":
		return ""
	case "Boolean":
		return "bool"
	case "String":
		return "string"
	default:
		panic("unknown native type " + t)
	}
}
