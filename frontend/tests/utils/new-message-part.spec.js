import { describe, it, expect } from 'vitest'
import { newPartOf, newMessagePartOf, normalizeText } from '@/utils/new-message-part'

describe('newPartOf — v0.23 шаг 5 (задача 378, новые письма)', () => {
  it('пусто → пусто', () => {
    expect(newPartOf('')).toBe('')
    expect(newPartOf('   ')).toBe('')
  })

  it('короткое письмо без истории → весь текст (кроме «-- ...» / подписи)', () => {
    const body = 'Алексей, сообщите сроки, когда ждать ответ. Ждём до конца недели.\nВика'
    expect(newPartOf(body)).toContain('сообщите сроки, когда ждать ответ')
    expect(newPartOf(body)).toContain('Ждём до конца недели')
  })

  it('письмо = новое + полная история предыдущего → только новое (diff сверху)', () => {
    const prev =
      'Поставила From: Отель Гранд Каньон [mailto:info@grandcanyon-hotel.ru]\n' +
      'Вот так сделать) на странице https://grandcanyon-hotel.ru/room/lyuks\n' +
      'Спасибо!'
    const body =
      'Виктория, а можете вместо этой, ту, что во вложении?\n' +
      'Выберем подходящую.\n\n' +
      prev +
      '\nПросьба при ответе на письмо сохранять историю переписки\n' +
      'С уважением,Маргарита Гребенюк'
    const newPart = newPartOf(body, prev)
    expect(newPart).toContain('вместо этой, ту, что во вложении')
    expect(newPart).toContain('Выберем подходящую')
    expect(newPart).not.toContain('Поставила From:')
    expect(newPart).not.toContain('С уважением,Маргарита')
    expect(newPart.length).toBeLessThan(body.length)
  })

  it('fallback без нового (idx===0): «...» + последние NEW_PART_TAIL знаков', () => {
    // prevBody совпадает с НАЧАЛОМ body → новое не найдено (idx===0), показывается хвост
    const long = 'А'.repeat(900) + ' конец письма'
    const out = newPartOf(long, long.slice(0, 100))
    expect(out).toContain('конец письма')
    expect(out.startsWith('…')).toBe(true)
  })

  it('обёртка newMessagePartOf == newPartOf', () => {
    const body = 'Тест новый текст\n\n-- Просьба\nС уважением'
    expect(newMessagePartOf(body)).toBe(newPartOf(body))
    expect(newMessagePartOf(body)).toContain('Тест новый текст')
    expect(newMessagePartOf(body)).not.toContain('-- Просьба')
    expect(newMessagePartOf(body)).not.toContain('С уважением')
  })

  it('normalizeText схлопывает пробелы/переносы', () => {
    expect(normalizeText('  a\n\n b\tc  ')).toBe('a b c')
    expect(normalizeText('')).toBe('')
  })

  it('MAX_NEW_PART ограничивает очень длинное «новое»', () => {
    const body = 'А'.repeat(9000) + '\nС уважением'
    const out = newPartOf(body)
    expect(out.length).toBeLessThanOrEqual(9000) // sanity
    expect(out).not.toContain('С уважением')
  })
})
