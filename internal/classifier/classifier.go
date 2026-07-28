// Package classifier предоставляет классификацию текста обращений:
// определение проекта, типа задачи и приоритета.
package classifier

import (
	"context"
)

// Classification содержит результат классификации.
type Classification struct {
	Project     string  `json:"project"`
	Type        string  `json:"type"`
	Priority    string  `json:"priority"`
	Confidence  float64 `json:"confidence"`
	NeedsTriage bool    `json:"needs_triage"`
}

// Classifier определяет интерфейс классификатора.
type Classifier interface {
	Classify(ctx context.Context, text string, projects, types []string) (*Classification, error)
}

// testRules возвращает правила для тестов.
func TestRules() []Rule {
	return []Rule{
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
