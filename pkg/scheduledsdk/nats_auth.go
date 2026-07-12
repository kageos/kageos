package scheduledsdk

import "github.com/kageos/kageos/pkg/controlauth"

const (
	TimerSchedulerMessageAuthScope  = "timer.scheduler.message.v1"
	TimerWorkerCommandAuthScope     = "timer.worker.command.v1"
	TimerSchedulerResponseAuthScope = "timer.scheduler.response.v1"
)

// SchedulerNATSAuth contains only the directions used by timer-scheduler.
type SchedulerNATSAuth struct {
	MessageSigner   *controlauth.Signer
	CommandVerifier *controlauth.Verifier
	ResponseSigner  *controlauth.Signer
}

func NewSchedulerNATSAuth(secret string) (*SchedulerNATSAuth, error) {
	messageSigner, err := controlauth.NewSigner(secret, TimerSchedulerMessageAuthScope)
	if err != nil {
		return nil, err
	}
	commandVerifier, err := controlauth.NewVerifier(secret, TimerWorkerCommandAuthScope, controlauth.VerifierOptions{})
	if err != nil {
		return nil, err
	}
	responseSigner, err := controlauth.NewSigner(secret, TimerSchedulerResponseAuthScope)
	if err != nil {
		return nil, err
	}
	return &SchedulerNATSAuth{
		MessageSigner:   messageSigner,
		CommandVerifier: commandVerifier,
		ResponseSigner:  responseSigner,
	}, nil
}

// WorkerNATSAuth contains only the directions used by trusted platform
// workers in app-server and agent-server. The root secret is never injected
// into a user App or the public SDK runtime.
type WorkerNATSAuth struct {
	MessageVerifier  *controlauth.Verifier
	CommandSigner    *controlauth.Signer
	ResponseVerifier *controlauth.Verifier
}

func NewWorkerNATSAuth(secret string) (*WorkerNATSAuth, error) {
	messageVerifier, err := controlauth.NewVerifier(secret, TimerSchedulerMessageAuthScope, controlauth.VerifierOptions{})
	if err != nil {
		return nil, err
	}
	commandSigner, err := controlauth.NewSigner(secret, TimerWorkerCommandAuthScope)
	if err != nil {
		return nil, err
	}
	responseVerifier, err := controlauth.NewVerifier(secret, TimerSchedulerResponseAuthScope, controlauth.VerifierOptions{})
	if err != nil {
		return nil, err
	}
	return &WorkerNATSAuth{
		MessageVerifier:  messageVerifier,
		CommandSigner:    commandSigner,
		ResponseVerifier: responseVerifier,
	}, nil
}
