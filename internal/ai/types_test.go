package ai_test

import (
	"encoding/json"
	"testing"

	"github.com/audetv/mailbridge/internal/ai"
)

func TestLLMResponse_Unmarshal(t *testing.T) {
	jsonStr := `{
		"verdicts": [
			{
				"action": "new",
				"task": {
					"title": "Обновить баннер",
					"description": "Клиент просит заменить баннер на главной",
					"priority": "medium",
					"project": "ТРК",
					"type": "content",
					"source_email_id": "msg-123"
				}
			},
			{
				"action": "update",
				"task_id": 42,
				"updates": {
					"priority": "urgent",
					"add_comment": "Клиент уточнил что проблема на проде"
				}
			},
			{
				"action": "completed",
				"task_id": 43,
				"comment": "Автор подтвердил решение"
			},
			{
				"action": "info_only",
				"summary": "Совещание перенесено"
			}
		]
	}`

	var resp ai.LLMResponse
	if err := json.Unmarshal([]byte(jsonStr), &resp); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if len(resp.Verdicts) != 4 {
		t.Errorf("expected 4 verdicts, got %d", len(resp.Verdicts))
	}

	if resp.Verdicts[0].Action != "new" {
		t.Errorf("Verdicts[0].Action = %s", resp.Verdicts[0].Action)
	}
	if resp.Verdicts[0].Task == nil {
		t.Error("Verdicts[0].Task is nil")
	}
	if resp.Verdicts[1].TaskID == nil || *resp.Verdicts[1].TaskID != 42 {
		t.Error("Verdicts[1].TaskID is not 42")
	}
}
