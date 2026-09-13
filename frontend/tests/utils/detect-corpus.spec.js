// @vitest-environment jsdom
// Корпусной тест детектора MD: все 191 комментарий из dev-БД (фикстура
// fixtures-corpus.json, сгенерированная из data/mailbridge.db).
// Инвариант: детектор классифицирует каждый body без исключений;
// MD-отчёты (##заголовки, списки, код) → true; короткие статусные
// комментарии → false.
import { describe, it, expect } from 'vitest'
import { looksLikeMd } from '@/utils/render-md'
import bodies from './fixtures-corpus.json'

describe('looksLikeMd — корпус из dev-БД (191 комментарий)', () => {
  it('классифицирует каждый body без исключений; MD-отчётов > 0', () => {
    expect(bodies.length).toBeGreaterThan(150)
    let md = 0
    for (const b of bodies) {
      const r = looksLikeMd(b)
      expect([true, false]).toContain(r)
      if (r) md++
    }
    expect(md).toBeGreaterThanOrEqual(5) // отчёты (карточка #49 и др.) есть
  })

  it('статусные комментарии (без MD) → false', () => {
    expect(looksLikeMd('Задача завершена')).toBe(false)
    expect(looksLikeMd('Задача закрыта: исполнитель подтвердил удаление фотографии по запросу')).toBe(false)
  })

  it('известный plain (карточка #49, c39 — ссылка на ТЗ) → false', () => {
    const b = bodies.find((x) => x.includes('Ссылка на ТЗ'))
    expect(b).toBeTruthy()
    expect(looksLikeMd(b)).toBe(false)
  })

  it('примеры MD-отчёта → true', () => {
    expect(looksLikeMd('# #49 — Gap-анализ сайта\n\n## 1. SEO-факторы\n**Технический аудит** сайта.\n\n- [ ] пункт\n\n`код`')).toBe(true)
  })
})
