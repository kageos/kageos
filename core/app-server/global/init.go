package global

// Init 初始化所有全局组件
func Init() {
	// 初始化 NATS 连接
	err := InitNats()
	if err != nil {
		panic(err)
	}
}

// Close 关闭所有全局组件
func Close() {
	CloseNats()
}
