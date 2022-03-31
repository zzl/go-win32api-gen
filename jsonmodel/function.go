package jsonmodel

type Function struct {
	Ns            *Api
	Name          string
	Name0         string
	SetLastError  bool
	DllImport     string
	ReturnType    *Type
	ReturnAttrs   []Attr
	Architectures []string
	Platform      string
	Attrs         []Attr
	Params        []struct {
		Name  string
		Type  *Type
		Attrs []Attr
	}
}
