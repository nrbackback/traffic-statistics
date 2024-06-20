package value_render

type oneLevelValueRender struct {
	field string
}

func NewOneLevelValueRender(template string) *oneLevelValueRender {
	return &oneLevelValueRender{
		field: template,
	}
}

func (vr *oneLevelValueRender) Render(event map[string]interface{}) interface{} {
	if value, ok := event[vr.field]; ok {
		return value
	}
	return nil
}
