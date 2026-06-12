package subjects

import "testing"

func TestTimerSubjectsUseExecutorKeySuffix(t *testing.T) {
	executorKey := "agent.session"
	if got, want := TimerExecutionRequestedSubject(executorKey), "timer.v1.cmd.execution.requested.agent.session"; got != want {
		t.Fatalf("TimerExecutionRequestedSubject() = %q, want %q", got, want)
	}
	if got, want := TimerWorkerQueueGroup(executorKey), "timer.worker.agent.session"; got != want {
		t.Fatalf("TimerWorkerQueueGroup() = %q, want %q", got, want)
	}
}

func TestNormalizeTimerSubjectSuffix(t *testing.T) {
	if got, want := NormalizeTimerSubjectSuffix(" App Function / Daily "), "app-function-daily"; got != want {
		t.Fatalf("NormalizeTimerSubjectSuffix() = %q, want %q", got, want)
	}
}

func TestTimerExecutionControlSubjects(t *testing.T) {
	tests := map[string]string{
		"started":   TimerExecutionStartedCommandSubject,
		"heartbeat": TimerExecutionHeartbeatCommandSubject,
		"finished":  TimerExecutionFinishedCommandSubject,
	}
	for name, subject := range tests {
		if subject == "" {
			t.Fatalf("%s subject is empty", name)
		}
	}
	if TimerExecutionControlQueueGroup == "" {
		t.Fatal("TimerExecutionControlQueueGroup is empty")
	}
}
