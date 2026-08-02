package bitnet

import (
	"testing"
)

func TestStreamFilterStripsBannerAndPromptEcho(t *testing.T) {
	filter := NewStreamFilter("The capital of France is")

	inputs := []string{
		"Loading model...\n",
		"\n",
		"▄▄ ▄▄\n",
		"██ ██\n",
		"██ ██  ▀▀█▄ ███▄███▄  ▀▀█▄    ▄████ ████▄ ████▄\n",
		"build      : b9918-390c30775\n",
		"model      : C:\\models\\ggml-model-i2_s.gguf\n",
		"available commands:\n",
		"  /exit or Ctrl+C     stop or exit\n",
		"\n",
		"> The capital of France is\n",
		"\n",
		"Paris is the capital of France.\n",
		"It is known as the City of Light.\n",
		"[ Prompt: 98.1 t/s | Generation: 19.8 t/s ]\n",
		"Exiting...\n",
		"> \n",
	}

	var output string
	for _, in := range inputs {
		res := filter.Filter(in)
		if res != "" {
			output += res
		}
	}

	expected := "Paris is the capital of France.\nIt is known as the City of Light.\n"
	if output != expected {
		t.Fatalf("expected:\n%q\ngot:\n%q", expected, output)
	}
}

func TestStreamFilterCleansSpecialTokens(t *testing.T) {
	filter := NewStreamFilter("Hello")
	res := filter.Filter("<|begin_of_text|>Hello world!<|end_of_text|>")
	if res != "Hello world!" {
		t.Fatalf("expected 'Hello world!', got %q", res)
	}
}

func TestCleanToken(t *testing.T) {
	tok := CleanToken("Hello <|end_of_text|>\r\n")
	if tok != "Hello \n" {
		t.Fatalf("expected 'Hello \\n', got %q", tok)
	}
}
