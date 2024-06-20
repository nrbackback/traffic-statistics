package codec

type Decoder interface {
	Decode(interface{}) map[string]interface{}
}

func NewDecoder(t string) Decoder {
	switch t {
	case "json_tag":
		return &structTagDecoder{
			tag: "json",
		}
	}
	return nil
}
