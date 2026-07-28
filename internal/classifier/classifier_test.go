package classifier_test

import (
	"context"
	"testing"

	"github.com/audetv/mailbridge/internal/classifier"
)

func TestClassifier_BugCritical(t *testing.T) {
	c := classifier.NewRuleBasedClassifier(classifier.DefaultRules())
	ctx := context.Background()

	result, err := c.Classify(ctx, "У нас не работает кабинет арендатора, ошибка 500", nil, nil)
	if err != nil {
		t.Fatalf("Classify error: %v", err)
	}

	if result.Type != "bug" {
		t.Errorf("expected type bug, got %s", result.Type)
	}
	if result.Project == "" {
		t.Error("expected project to be detected")
	}
}

func TestClassifier_AccessRequest(t *testing.T) {
	c := classifier.NewRuleBasedClassifier(classifier.DefaultRules())
	ctx := context.Background()

	result, err := c.Classify(ctx, "Нужен доступ к админке театра для нового сотрудника", nil, nil)
	if err != nil {
		t.Fatalf("Classify error: %v", err)
	}

	if result.Type != "access" {
		t.Errorf("expected type access, got %s", result.Type)
	}
	if result.Project != "" {
		t.Logf("detected project: %s", result.Project)
	}
}

func TestClassifier_UrgencyBoost(t *testing.T) {
	c := classifier.NewRuleBasedClassifier(classifier.DefaultRules())
	ctx := context.Background()

	result, err := c.Classify(ctx, "Срочно! Сайт отеля упал, клиенты не могут забронировать!", nil, nil)
	if err != nil {
		t.Fatalf("Classify error: %v", err)
	}

	if result.Priority != "urgent" {
		t.Errorf("expected priority urgent, got %s", result.Priority)
	}
}

func TestClassifier_ContentUpdate(t *testing.T) {
	c := classifier.NewRuleBasedClassifier(classifier.DefaultRules())
	ctx := context.Background()

	result, err := c.Classify(ctx, "Обновите пожалуйста баннер на главной странице ТРК", nil, nil)
	if err != nil {
		t.Fatalf("Classify error: %v", err)
	}

	if result.Type != "content" {
		t.Errorf("expected type content, got %s", result.Type)
	}
}

func TestClassifier_ManualFields(t *testing.T) {
	c := classifier.NewRuleBasedClassifier(classifier.DefaultRules())
	ctx := context.Background()

	text := "Проект: Отель\nТип: bug\nПриоритет: high\n\nНе работает форма бронирования"

	result, err := c.Classify(ctx, text, nil, nil)
	if err != nil {
		t.Fatalf("Classify error: %v", err)
	}

	if result.Project != "Отель" {
		t.Errorf("expected project Отель, got %s", result.Project)
	}
	if result.Type != "bug" {
		t.Errorf("expected type bug, got %s", result.Type)
	}
	if result.Priority != "High" {
		t.Errorf("expected priority High, got %s", result.Priority)
	}
	if result.Confidence < 0.9 {
		t.Errorf("expected high confidence, got %f", result.Confidence)
	}
}

func TestClassifier_EmptyText(t *testing.T) {
	c := classifier.NewRuleBasedClassifier(classifier.DefaultRules())
	ctx := context.Background()

	result, err := c.Classify(ctx, "", nil, nil)
	if err != nil {
		t.Fatalf("Classify error: %v", err)
	}

	if !result.NeedsTriage {
		t.Error("empty text should need triage")
	}
}

func TestClassifier_SEO(t *testing.T) {
	c := classifier.NewRuleBasedClassifier(classifier.DefaultRules())
	ctx := context.Background()

	result, err := c.Classify(ctx, "Нужно провести SEO оптимизацию для сайта отеля, проверить метатеги", nil, nil)
	if err != nil {
		t.Fatalf("Classify error: %v", err)
	}

	if result.Type != "seo" {
		t.Errorf("expected type seo, got %s", result.Type)
	}
}

func TestClassifier_HotelProject(t *testing.T) {
	c := classifier.NewRuleBasedClassifier(classifier.DefaultRules())
	ctx := context.Background()

	result, err := c.Classify(ctx, "Не работает бронирование номеров в отеле", nil, nil)
	if err != nil {
		t.Fatalf("Classify error: %v", err)
	}

	if result.Project != "Отель" {
		t.Errorf("expected project Отель, got %s", result.Project)
	}
}

func TestClassifier_FitnessProject(t *testing.T) {
	c := classifier.NewRuleBasedClassifier(classifier.DefaultRules())
	ctx := context.Background()

	result, err := c.Classify(ctx, "В фитнес клубе не отображается расписание", nil, nil)
	if err != nil {
		t.Fatalf("Classify error: %v", err)
	}

	if result.Project != "Фитнес-клуб" {
		t.Errorf("expected project Фитнес-клуб, got %s", result.Project)
	}
}

func TestClassifier_DefaultPriority(t *testing.T) {
	c := classifier.NewRuleBasedClassifier(classifier.DefaultRules())
	ctx := context.Background()

	result, err := c.Classify(ctx, "Подскажите как добавить пользователя в систему", nil, nil)
	if err != nil {
		t.Fatalf("Classify error: %v", err)
	}

	if result.Priority != "medium" {
		t.Errorf("expected default priority medium, got %s", result.Priority)
	}
}
