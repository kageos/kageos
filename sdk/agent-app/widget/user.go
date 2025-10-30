package widget

type User struct {
}

func (u *User) Config() interface{} {
	return u
}

func (u *User) Type() string {
	return TypeUser
}

func newUser(widgetParsed map[string]string) *User {
	user := &User{}
	return user
}
