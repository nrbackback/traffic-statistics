package value_render

type boolValueRender struct {
	value bool
}

func NewBoolValueRender(value bool) *boolValueRender {
	return &boolValueRender{
		value: value,
	}
}

func (vr *boolValueRender) Render(event map[string]interface{}) interface{} {
	return vr.value
}
