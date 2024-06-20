package value_render

import (
	"bytes"
	"text/template"

	"traffic-statistics/pkg/log"
)

type templateValueRender struct {
	tmpl *template.Template
}

func newTemplateValueRender(t string) *templateValueRender {
	tmpl, err := template.New(t).Parse(t)
	if err != nil {
		log.Fatalw("could not parse template", "template", t, "error", err)
	}
	return &templateValueRender{
		tmpl: tmpl,
	}
}

func (r *templateValueRender) Render(event map[string]interface{}) interface{} {
	b := bytes.NewBuffer(nil)
	if r.tmpl.Execute(b, event) != nil {
		return nil
	}
	return string(b.Bytes())
}
