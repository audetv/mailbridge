package ai_test

import (
	"context"
)

type mockClient struct {
	response string
	err      error
}

func (m *mockClient) Generate(_ context.Context, _ string, _ []string) (string, error) {
	return m.response, m.err
}

// func TestDetermineThreadID(t *testing.T) {
// 	email := &extractor.ExtractedEmail{
// 		MessageID:  "msg-2",
// 		References: []string{"msg-1", "msg-0"},
// 	}

// 	threadID := ai.DetermineThreadIDForTest(email)
// 	if threadID != "msg-1" {
// 		t.Errorf("threadID = %s, want msg-1", threadID)
// 	}
// }
