package subjects

const (
	// message / gateway
	MessageSendCommandSubject                 = "message.v1.cmd.send"
	MessageSendQueueGroup                     = MessageSendCommandSubject
	GatewayTokenInvalidateCommandSubject      = "gateway.v1.cmd.token.invalidate"
	GatewayTokenRemoveBlacklistCommandSubject = "gateway.v1.cmd.token.remove-blacklist"
)
