package widget

type Text struct {
	Format string `json:"format"` //json，yaml，xml，markdown，html，csv 等等
}

func (i *Text) Config() interface{} {
	return i
}

func (i *Text) Type() string {
	return TypeText
}
