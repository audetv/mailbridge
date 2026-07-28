package classifier_test

import (
	"context"
	"testing"

	"github.com/audetv/mailbridge/internal/classifier"
)

// testClassifier создаёт RuleBasedClassifier с тестовыми правилами.
func testClassifier() *classifier.RuleBasedClassifier {
	return classifier.NewRuleBasedClassifier(
		testRules(),
		[]string{"срочно", "горит", "упал", "asap", "urgent", "всё сломалось", "аврал", "пожар"},
		map[string]bool{
			"bug": true, "feature": true, "support": true,
			"access": true, "seo": true, "content": true,
		},
		map[string]bool{
			"urgent": true, "high": true, "medium": true, "low": true,
		},
	)
}

// testRules возвращает правила для тестов.
func testRules() []classifier.Rule {
	return []classifier.Rule{
		{
			Keywords: []string{"не работает", "упал", "ошибка сервер", "внутренняя ошибка", "502", "503"},
			Type:     "bug",
			Priority: "urgent",
			Weight:   5,
		},
		{
			Keywords: []string{"ошибка", "баг", "некорректно", "глючит", "не отображается", "не открывается", "не загружается"},
			Type:     "bug",
			Priority: "high",
			Weight:   3,
		},
		{
			Keywords: []string{"доступ", "пароль", "логин", "завести пользователь", "права доступ", "авторизация не проходит", "не могу войти"},
			Type:     "access",
			Priority: "medium",
			Weight:   3,
		},
		{
			Keywords: []string{"seo", "продвижение", "поисковая оптимизация", "яндекс метрика", "метатег", "поисковая выдача", "семантика"},
			Type:     "seo",
			Priority: "medium",
			Weight:   2,
		},
		{
			Keywords: []string{"обновить баннер", "поменять текст", "добавить новость", "акция", "обновить информацию", "заменить фото", "добавить страницу"},
			Type:     "content",
			Priority: "low",
			Weight:   2,
		},
		{
			Keywords: []string{"консультация", "вопрос", "как сделать", "объясните", "помогите", "подскажите"},
			Type:     "support",
			Priority: "medium",
			Weight:   1,
		},
		{
			Keywords: []string{"трк", "арендатор", "торговый комплекс", "кабинет арендатора"},
			Project:  "ТРК",
			Weight:   2,
		},
		{
			Keywords: []string{"отель", "гостиница", "номер", "бронирование"},
			Project:  "Отель",
			Weight:   2,
		},
		{
			Keywords: []string{"фитнес", "клуб", "тренажерный зал"},
			Project:  "Фитнес-клуб",
			Weight:   2,
		},
		{
			Keywords: []string{"театр", "билет", "спектакль", "yoomoney"},
			Project:  "Театр",
			Weight:   2,
		},
		{
			Keywords: []string{"мебель", "мебельный центр", "маркетплейс"},
			Project:  "Мебельный центр",
			Weight:   2,
		},
		{
			Keywords: []string{"склад", "складской комплекс"},
			Project:  "Складской комплекс",
			Weight:   2,
		},
		{
			Keywords: []string{"кафе", "ресторан", "меню"},
			Project:  "Кафе",
			Weight:   2,
		},
		{
			Keywords: []string{"арена", "каток", "расписание лед"},
			Project:  "Ледовая арена",
			Weight:   2,
		},
		{
			Keywords: []string{"визитка", "корпоративный сайт", "публичный документ"},
			Project:  "Корпоративные сайты",
			Weight:   2,
		},
	}
}

func TestClassifier_BugCritical(t *testing.T) {
	c := testClassifier()
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
	c := testClassifier()
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
	c := testClassifier()
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
	c := testClassifier()
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
	c := testClassifier()
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
	c := testClassifier()
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
	c := testClassifier()
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
	c := testClassifier()
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
	c := testClassifier()
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
	c := testClassifier()
	ctx := context.Background()

	result, err := c.Classify(ctx, "Подскажите как добавить пользователя в систему", nil, nil)
	if err != nil {
		t.Fatalf("Classify error: %v", err)
	}

	if result.Priority != "medium" {
		t.Errorf("expected default priority medium, got %s", result.Priority)
	}
}
