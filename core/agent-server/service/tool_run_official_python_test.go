package service

import "testing"

func TestNormalizeOfficialPythonCode(t *testing.T) {
	input := "\ufeffdef agentos_entry(args, output_dir):\r\n\tinp\x1b[32;1:3uut_files = args['input_files']\x00\r\n\treturn {'data': input_files}\n"

	got := normalizeOfficialPythonCode(input)
	want := "def agentos_entry(args, output_dir):\r\n\tinp\x1b[32;1:3uut_files = args['input_files']\x00\r\n\treturn {'data': input_files}\n"
	if got != want {
		t.Fatalf("normalizeOfficialPythonCode() = %q, want %q", got, want)
	}
}
