package sender_test

import (
	"testing"

	"github.com/audetv/mailbridge/internal/sender"
)

func TestFormatAcknowledgement(t *testing.T) {
	data := &sender.AcknowledgementData{
		To:                 "user@example.com",
		Subject:            "Не работает сайт",
		InReplyToMessageID: "original-msg-id",
		IssueSequence:      "WEB-123",
		ProjectName:        "ТРК",
		TypeName:           "bug",
		Priority:           "high",
	}

	subject, body := sender.FormatAcknowledgement(data)

	if subject != "Re: [WEB-123] Не работает сайт" {
		t.Errorf("subject = %q", subject)
	}
	if body == "" {
		t.Error("body is empty")
	}
	if !contains(body, "WEB-123") {
		t.Error("body doesn't contain issue sequence")
	}
	if !contains(body, "MAILBRIDGE-INTERNAL") {
		t.Error("body doesn't contain internal marker")
	}
}

func TestFormatCommentReply(t *testing.T) {
	data := &sender.CommentReplyData{
		To:                 "user@example.com",
		Subject:            "Не работает сайт",
		InReplyToMessageID: "original-msg-id",
		References:         []string{"ref-1", "ref-2"},
		IssueSequence:      "WEB-123",
		CommentText:        "Проверил, проблема с сертификатом",
		CommentAuthor:      "Руслан",
	}

	subject, body := sender.FormatCommentReply(data)

	if subject != "Re: [WEB-123] Не работает сайт" {
		t.Errorf("subject = %q", subject)
	}
	if !contains(body, "Руслан") {
		t.Error("body doesn't contain author name")
	}
	if !contains(body, "Проверил") {
		t.Error("body doesn't contain comment text")
	}
}

func TestFormatStatusChange(t *testing.T) {
	data := &sender.StatusChangeData{
		To:                 "user@example.com",
		Subject:            "Не работает сайт",
		InReplyToMessageID: "original-msg-id",
		IssueSequence:      "WEB-123",
		OldStatus:          "in_progress",
		NewStatus:          "completed",
	}

	subject, body := sender.FormatStatusChange(data)

	if subject != "Re: [WEB-123] Не работает сайт" {
		t.Errorf("subject = %q", subject)
	}
	if !contains(body, "in_progress") || !contains(body, "completed") {
		t.Error("body doesn't contain statuses")
	}
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && len(s) >= len(substr) && stringContains(s, substr)
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
