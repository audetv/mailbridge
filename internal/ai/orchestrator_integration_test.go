package ai_test

// func TestOrchestrator_IntegrationWithOllama(t *testing.T) {
// 	// Создаём реальный клиент к Ollama
// 	client := ai.NewOllamaClient("http://localhost:11434", "email-assistant")

// 	// Создаём in-memory БД
// 	st, _ := sqlite.NewStore(":memory:")
// 	_ = st.Migrate(context.Background())
// 	defer st.Close()

// 	orch := ai.NewOrchestrator(client, st)

// 	email := &extractor.ExtractedEmail{
// 		MessageID:  "test-msg-001",
// 		From:       "client@example.com",
// 		Subject:    "Ошибка на сайте",
// 		BodyText:   "From: ivan@company.com\nMessage-ID: <urgent-site-down-001@company.com>\nSubject: Срочно! Сайт упал\nТекст: Коллеги, главный сайт недоступен, выдает 502. Скриншот ошибки прикладываю. Нужно срочно поднять, клиенты звонят.\n[ИЗОБРАЖЕНИЕ: base64_string_here_или_пусто_для_теста]",
// 		References: []string{"thread-001"},
// 	}

// 	resp, err := orch.ProcessEmail(context.Background(), email)
// 	if err != nil {
// 		t.Fatalf("ProcessEmail error: %v", err)
// 	}

// 	t.Logf("Verdicts: %+v", resp)

// 	if err := orch.ApplyVerdicts(context.Background(), email, resp); err != nil {
// 		t.Fatalf("ApplyVerdicts error: %v", err)
// 	}

// 	tasks, _ := st.GetActiveTasksByThread(context.Background(), "thread-001")
// 	t.Logf("Created tasks: %d", len(tasks))
// 	for _, task := range tasks {
// 		t.Logf("  Task #%d: %s (проект: %s, приоритет: %s)", task.ID, task.Subject, task.Project, task.Priority)
// 	}
// }
