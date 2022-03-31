package utils

import (
	"strconv"
	"strings"
	"unsafe"
)

const PtrSize = int(unsafe.Sizeof(uintptr(0)))

func CapName(name string) string {
	var c uint8
	for {
		c = name[0]
		if c != '_' {
			break
		}
		name = name[1:] + "_"
	}
	if c >= 'a' && c <= 'z' {
		name = string(c-32) + name[1:]
	}
	name = strings.Replace(name, "_e__Union", "", 1)
	name = strings.Replace(name, "_e__Struct", "", 1)
	return name
}

func SafeGoName(name string) string {
	reservedNames := []string{"type", "var", "range", "map"}
	for _, it := range reservedNames {
		if name == it {
			return name + "_"
		}
	}
	return name
}

type SizeInfo struct {
	TotalSize int
	AlignSize int
}

func (me SizeInfo) String() string {
	return strconv.Itoa(me.TotalSize) + "(" + strconv.Itoa(me.AlignSize) + ")"
}

func StructSize(fieldSizes ...SizeInfo) SizeInfo {
	sumSize := 0
	maxAlignSize := 0
	for _, size := range fieldSizes {
		if size.AlignSize == 0 {
			size.AlignSize = size.TotalSize
		}
		if sumSize%size.AlignSize != 0 {
			sumSize += size.AlignSize - sumSize%size.AlignSize
		}
		if size.AlignSize > maxAlignSize {
			maxAlignSize = size.AlignSize
		}
		sumSize += size.TotalSize
	}
	if sumSize != 0 && sumSize%maxAlignSize != 0 {
		sumSize += maxAlignSize - sumSize%maxAlignSize
	}
	return SizeInfo{sumSize, maxAlignSize}
}

func BuildGuidExpr(sGuid string) string {
	expr := "syscall.GUID{0x" + sGuid[:8] +
		", 0x" + sGuid[9:13] + ", 0x" + sGuid[14:18] + ", \n\t[8]byte{"
	sGuid = strings.Replace(sGuid[19:], "-", "", 1)
	for n := 0; n < 16; n += 2 {
		if n > 0 {
			expr += ", "
		}
		expr += "0x" + sGuid[n:n+2]
	}
	expr += "}}"
	return expr
}
