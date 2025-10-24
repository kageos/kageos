package app

import (
	"strings"
)

func routerKey(router string, method string) string {
	router = strings.Trim(router, "/")
	key := router + "." + method
	return key
}

func register(router string, method string, handleFunc HandleFunc, templater Templater) {
	if app == nil {
		initApp()
	}
	//router = strings.Trim(router, "/")
	//key := router + "." + method
	app.routerInfo[routerKey(router, method)] = &routerInfo{
		HandleFunc: handleFunc,
		Router:     router,
		Method:     method,
		Template:   templater,
	}
}

func GET(router string, handleFunc HandleFunc, templater Templater) {
	register(router, "GET", handleFunc, templater)
}

func POST(router string, handleFunc HandleFunc, templater Templater) {
	register(router, "POST", handleFunc, templater)
}
func PUT(router string, handleFunc HandleFunc, templater Templater) {
	register(router, "PUT", handleFunc, templater)
}

func DELETE(router string, handleFunc HandleFunc, templater Templater) {
	register(router, "DELETE", handleFunc, templater)
}
