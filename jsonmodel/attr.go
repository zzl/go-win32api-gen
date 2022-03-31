package jsonmodel

import (
	"encoding/json"
	"fmt"
)

type Attr struct {
	Props map[string]interface{}
	Str   string
}

func (this *Attr) UnmarshalJSON(p []byte) error {
	s := string(p)
	if s[0] == '{' {
		props := make(map[string]interface{})
		e := json.Unmarshal(p, &props)
		if e != nil {
			panic(e)
		}
		this.Props = props
	} else {
		this.Str = s[1 : len(s)-1]
	}
	return nil
}

func (this *Attr) String() string {
	if this.Props == nil {
		return this.Str
	}
	s := ""
	for k, v := range this.Props {
		if s != "" {
			s += ", "
		}
		s += k + ": "
		s += fmt.Sprintf("%v", v)
	}
	return s
}

func BuildAttrsStr(attrs []Attr) string {
	s := ""
	for n, a := range attrs {
		if n > 0 {
			s += ", "
		}
		s += a.String()
	}
	return s
}
