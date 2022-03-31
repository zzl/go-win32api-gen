package jsonmodel

import "math/big"

type Constant struct {
	Name      string
	Type      *Type
	ValueType string
	Value     ConstantValue
	Attrs     []Attr
}

type ConstantValue struct {
	Int *big.Int
	Str string
}

func (this *ConstantValue) UnmarshalJSON(p []byte) error {
	s := string(p)
	var n big.Int
	_, ok := n.SetString(s, 10)
	if ok {
		this.Int = &n
	} else {
		if s[0] == '"' {
			s = s[1 : len(s)-1]
		}
		this.Str = s
	}
	return nil
}

func (this *ConstantValue) String() string {
	if this.Int != nil {
		return this.Int.String()
	} else {
		return this.Str
	}
}
