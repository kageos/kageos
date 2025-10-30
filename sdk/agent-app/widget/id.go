package widget

type ID struct {
}

func (i *ID) Config() interface{} {
	return i
}

func (i *ID) Type() string {
	return TypeID
}

func newID(widgetParsed map[string]string) *ID {
	id := &ID{}

	return id
}
