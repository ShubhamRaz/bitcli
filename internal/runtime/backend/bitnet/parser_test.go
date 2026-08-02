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

func TestStreamFilterStopsAtEotID(t *testing.T) {
	filter := NewStreamFilter("test prompt")
	// Simulate the filter seeing a header, then response, then eot_id.
	filter.Filter("Loading model...\n")
	filter.Filter("> test prompt\n")
	res1 := filter.Filter("Paris is great.\n")
	res2 := filter.Filter("More info.<|eot_id|>\n")
	res3 := filter.Filter("This should not appear.\n")

	if res1 != "Paris is great.\n" {
		t.Fatalf("expected first line, got %q", res1)
	}
	if res2 != "More info." {
		t.Fatalf("expected text before eot_id, got %q", res2)
	}
	if res3 != "" {
		t.Fatalf("expected empty after eot_id, got %q", res3)
	}
}

func TestStreamFilterStripsMultiTurnChatEcho(t *testing.T) {
	chatPrompt := "<|begin_of_text|>System: You are a helpful, concise, and friendly AI assistant. Answer the user's questions clearly and accurately.<|eot_id|>\nUser: hello\nAssistant:"
	filter := NewStreamFilter(chatPrompt)

	inputs := []string{
		"Loading model...\n",
		"\n",
		"build      : b9918-390c30775\n",
		"model      : C:\\models\\ggml-model-i2_s.gguf\n",
		"available commands:\n",
		"  /exit or Ctrl+C     stop or exit\n",
		"\n",
		"> <|begin_of_text|>System: You are a helpful, concise, and friendly AI assistant. Answer the user's questions clearly and accurately.<|eot_id|>\n",
		"User: hello<|eot_id|>\n",
		"Assistant:\n",
		"\n",
		"Hello! How can I help you today?\n",
		"Exiting...\n",
	}

	var output string
	for _, in := range inputs {
		res := filter.Filter(in)
		if res != "" {
			output += res
		}
	}

	expected := "Hello! How can I help you today?\n"
	if output != expected {
		t.Fatalf("expected:\n%q\ngot:\n%q", expected, output)
	}
}

