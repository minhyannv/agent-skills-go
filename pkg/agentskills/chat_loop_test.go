package agentskills

import "testing"

func TestToOpenAIMessagesAddsSystemWhenMissing(t *testing.T) {
	app := &App{systemPrompt: "system prompt"}

	out, err := app.toOpenAIMessages([]Message{
		{Role: RoleUser, Content: "hello"},
	})
	if err != nil {
		t.Fatalf("toOpenAIMessages returned error: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 messages (system + user), got %d", len(out))
	}
}

func TestToOpenAIMessagesWithSystemDoesNotDuplicate(t *testing.T) {
	app := &App{systemPrompt: "system prompt"}

	out, err := app.toOpenAIMessages([]Message{
		{Role: RoleSystem, Content: "custom system"},
		{Role: RoleUser, Content: "hello"},
	})
	if err != nil {
		t.Fatalf("toOpenAIMessages returned error: %v", err)
	}
	if len(out) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(out))
	}
}

func TestToOpenAIMessagesRejectsInvalidRole(t *testing.T) {
	app := &App{systemPrompt: "system prompt"}

	_, err := app.toOpenAIMessages([]Message{
		{Role: "tool", Content: "bad"},
	})
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
}
