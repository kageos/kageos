package widget

type Switch struct {

	//大道至简，先mvp产品，无关紧要的字段都先砍掉
}

func (s *Switch) Config() interface{} {
	return s
}

func (s *Switch) Type() string {
	return TypeSwitch
}

func newSwitch(widgetParsed map[string]string) *Switch {
	switchWidget := &Switch{}
	return switchWidget
}
