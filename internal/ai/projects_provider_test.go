package ai_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/audetv/mailbridge/internal/ai"
	"github.com/audetv/mailbridge/internal/extractor"
	"github.com/audetv/mailbridge/internal/store"
	"github.com/audetv/mailbridge/internal/store/sqlite"
)

func testEmail() *extractor.ExtractedEmail {
	return &extractor.ExtractedEmail{
		From:     "user@example.com",
		Subject:  "Тестовое письмо",
		BodyText: "Текст письма",
	}
}

// Провайдер проектов (шаг 14): статический список не должен просочиться
// в промпт, когда провайдер вернул нормальный список.
func TestBuildPrompt_ProjectsFromProvider(t *testing.T) {
	o := ai.NewOrchestrator(nil, nil)
	o.SetProjects([]string{"Старый"})
	o.SetProjectsProvider(func(_ context.Context) ([]string, error) {
		return []string{"ТРК", ai.DefaultProject}, nil
	})

	prompt := o.BuildPrompt("Резюме", []*store.Task{}, testEmail())

	for _, want := range []string{"=== ДОСТУПНЫЕ ПРОЕКТЫ ===", "ТРК", ai.DefaultProject} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
	if strings.Contains(prompt, "Старый") {
		t.Error("static projects leaked into prompt; provider must win")
	}
}

// Провайдер с ошибкой — оркестратор не падает, откатывается на статику.
func TestProjectsProvider_ErrorFallsBackToStatic(t *testing.T) {
	o := ai.NewOrchestrator(nil, nil)
	o.SetProjects([]string{"Отель"})
	o.SetProjectsProvider(func(_ context.Context) ([]string, error) {
		return nil, errors.New("db down")
	})

	prompt := o.BuildPrompt("Резюме", nil, testEmail())
	if !strings.Contains(prompt, "Отель") {
		t.Error("static projects not used on provider error")
	}
}

// Провайдер вернул пустой список (БД пуста) — fallback-проект всегда доступен для AI.
func TestProjectsProvider_EmptyListIncludesDefault(t *testing.T) {
	o := ai.NewOrchestrator(nil, nil)
	o.SetProjectsProvider(func(_ context.Context) ([]string, error) {
		return []string{ai.DefaultProject}, nil
	})

	prompt := o.BuildPrompt("Резюме", nil, testEmail())
	if !strings.Contains(prompt, ai.DefaultProject) {
		t.Error("default project missing from empty-provider prompt")
	}
}

// Шаг 14+15: проект из вердикта — валидация против активных проектов;
// пустой/неизвестный → fallback на ai.DefaultProject (MAILBRIDGE_DEFAULT_PROJECT).
func TestResolveVerdictProject(t *testing.T) {
	cases := []struct {
		name     string
		verdict  string
		provider []string
		want     string
	}{
		{name: "пустой вердикт — fallback", verdict: "", provider: []string{"ТРК"}, want: ai.DefaultProject},
		{name: "пробелы — fallback", verdict: "   ", provider: []string{"ТРК"}, want: ai.DefaultProject},
		{name: "известный проект сохраняется", verdict: "ТРК", provider: []string{"ТРК", "Отель"}, want: "ТРК"},
		{name: "неизвестный проект — fallback", verdict: "Закрытый проект", provider: []string{"ТРК"}, want: ai.DefaultProject},
		{name: "пробелы вокруг известного — trim до вхождения", verdict: " ТРК ", provider: []string{"ТРК"}, want: "ТРК"},
		{name: "fallback сам по себе разрешается", verdict: ai.DefaultProject, provider: []string{ai.DefaultProject}, want: ai.DefaultProject},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			o := ai.NewOrchestrator(nil, nil)
			o.SetProjectsProvider(func(_ context.Context) ([]string, error) {
				return tc.provider, nil
			})
			got := o.ResolveVerdictProject(context.Background(), tc.verdict)
			if got != tc.want {
				t.Errorf("ResolveVerdictProject(%q) = %q, want %q", tc.verdict, got, tc.want)
			}
		})
	}
}

// End-to-end: ApplyVerdicts с неизвестным проектом пишет в БД fallback-проект
// (регрессия бага 4 — «Проект: не найдено» по AI-созданным задачам).
func TestApplyVerdicts_UnknownProjectFallsBackToDefault(t *testing.T) {
	ctx := context.Background()
	st, err := sqlite.NewStore(":memory:")
	if err != nil {
		t.Fatalf("NewStore: %v", err)
	}
	_ = st.Migrate(ctx)
	defer st.Close()

	o := ai.NewOrchestrator(nil, st)
	o.SetProjectsProvider(func(_ context.Context) ([]string, error) {
		return []string{"ТРК"}, nil
	})

	email := &extractor.ExtractedEmail{
		MessageID: "msg-x1",
		From:      "user@example.com",
		Subject:   "Test",
		BodyText:  "Body",
	}

	unknown := &ai.LLMResponse{
		Verdicts: []ai.Verdict{
			{
				Action: "new",
				Task: &ai.NewTaskData{
					Title:   "Задача с неизвестным проектом",
					Project: "Придуманный проект",
					Type:    "bug",
				},
			},
		},
	}
	if err := o.ApplyVerdicts(ctx, email, unknown, 0); err != nil {
		t.Fatalf("ApplyVerdicts: %v", err)
	}

	result, err := st.ListTasks(ctx, &store.TaskFilter{Page: 1, PerPage: 10})
	if err != nil {
		t.Fatalf("ListTasks: %v", err)
	}
	if len(result.Tasks) != 1 {
		t.Fatalf("expected 1 task in db, got %d", len(result.Tasks))
	}
	gotProject := result.Tasks[0].Project
	if gotProject != ai.DefaultProject {
		t.Errorf("task.project = %q, want fallback %q", gotProject, ai.DefaultProject)
	}
}
