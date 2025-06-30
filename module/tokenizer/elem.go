package tokenizer

import (
	"fmt"
	"strconv"
)

const (
	Str Kind = iota
	Ident
)

type Kind uint8

type Elem interface {
	Kind() Kind
	Expr() string
	Content() string

	Str(key string) (string, bool)
	Int(key string) (int64, bool)
	Boolean(key string) (bool, bool)

	String() string
}

type strElem struct {
	kind    Kind
	content string
}

type nodeElem struct {
	strElem

	count int

	expr       string
	attributes map[string]string
}

var (
	_ Elem = (*strElem)(nil)
	_ Elem = (*nodeElem)(nil)
)

func (s strElem) Kind() Kind                  { return s.kind }
func (s strElem) Content() string             { return s.content }
func (s strElem) String() string              { return s.content }
func (s strElem) Expr() string                { panic("implement me") }
func (s strElem) Str(string) (string, bool)   { panic("implement me") }
func (s strElem) Int(string) (int64, bool)    { panic("implement me") }
func (s strElem) Boolean(string) (bool, bool) { panic("implement me") }

func (s nodeElem) Expr() string { return s.expr }
func (s nodeElem) String() string {
	attr := ""
	for k, v := range s.attributes {
		if attr == "" {
			attr += " "
		}
		attr += k + "=" + v
	}
	if s.content == "" {
		return fmt.Sprintf("<%s%s />", s.expr, attr)
	}
	return fmt.Sprintf("<%s%s>%s</%s>", s.expr, attr, s.content, s.expr)
}

func (s nodeElem) Str(key string) (string, bool) {
	value, ok := s.attributes[key]
	if !ok {
		return "", false
	}
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return value, true
	}
	return value[1 : len(value)-1], true
}

func (s nodeElem) Int(key string) (int64, bool) {
	value, ok := s.attributes[key]
	if !ok {
		return 0, false
	}
	i, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return i, false
	}
	return i, true
}

func (s nodeElem) Boolean(key string) (bool, bool) {
	value, ok := s.attributes[key]
	if !ok {
		return false, false
	}
	if value == "" {
		return true, true
	}
	i, err := strconv.ParseBool(value)
	if err != nil {
		return i, false
	}
	return i, true
}
