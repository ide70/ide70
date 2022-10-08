package api

import (
	"strings"
	"text/template"
)

type Text struct {
}

func (a *API) Text() *Text {
	return &Text{}
}

type Template struct {
	t *template.Template
}

func (text *Text) CreateTemplate(body string) *Template {
	t := &Template{}
	templ, err := template.New("").Parse(body)
	if err != nil {
		logger.Error("Parse template", err.Error())
		return nil
	}
	t.t = templ
	return t
}

func (t *Template) Render(data interface{}) string {
	sb := &strings.Builder{}
	t.t.Execute(sb, data)
	return sb.String()
}

func (t *Template) RenderBinary(data interface{}) *BinaryData {
	sb := &strings.Builder{}
	t.t.Execute(sb, data)
	res := []byte(sb.String())
	return &BinaryData{data: &res}
}
