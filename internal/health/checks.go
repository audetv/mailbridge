// Package health предоставляет проверки состояния компонентов.
package health

import "context"

// Checker описывает компонент, который может проверить своё состояние.
type Checker interface {
	Name() string
	Check(ctx context.Context) error
}

// NamedCheck создаёт простую проверку с именем и функцией.
type NamedCheck struct {
	name  string
	check func(ctx context.Context) error
}

// NewNamedCheck создаёт новый NamedCheck.
func NewNamedCheck(name string, check func(ctx context.Context) error) *NamedCheck {
	return &NamedCheck{
		name:  name,
		check: check,
	}
}

// Name возвращает имя проверки.
func (n *NamedCheck) Name() string {
	return n.name
}

// Check выполняет проверку.
func (n *NamedCheck) Check(ctx context.Context) error {
	return n.check(ctx)
}
