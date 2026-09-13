import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mdToPlainText, copyComment } from '@/utils/copy-comment'

describe('mdToPlainText', () => {
  it('plain text — без изменений (переносы сохранены)', () => {
    const t = 'Здравствуйте,\nPavel!\n\nПриложу договор.'
    expect(mdToPlainText(t)).toBe(t)
  })

  it('жирный/курсив: **x** __y__ *z* → x y z', () => {
    expect(mdToPlainText('**жирный** и __двойной__ и *звезды*')).toBe(
      'жирный и двойной и звезды'
    )
  })

  it('заголовки: ## Статус → Статус', () => {
    expect(mdToPlainText('## Статус задачи\n\ntext')).toBe('Статус задачи\n\ntext')
  })

  it('ссылки: [Текст](http://x) → Текст', () => {
    expect(mdToPlainText('См. [Текст](http://example.com/page) дальше')).toBe(
      'См. Текст дальше'
    )
  })

  it('маркированные списки — буллеты', () => {
    const t = mdToPlainText('- пункт один\n- пункт два\n1. нумерованный\n')
    expect(t).toBe('• пункт один\n• пункт два\n• нумерованный')
  })

  it('цитаты: > текст → текст; строки не липнут', () => {
    expect(mdToPlainText('> quoted line\nnext')).toBe('quoted line\nnext')
  })

  it('код-обёртка: `код` → код', () => {
    expect(mdToPlainText('используй `npm ci` локально')).toBe('используй npm ci локально')
  })

  it('realtalk-пример из dev-БД (reply/отчёт с ** и ##)', () => {
    const t = mdToPlainText('## Статус задачи #40 (02.09)\n\n**Выполнено:** сайт опубликован.\n')
    expect(t).toContain('Статус задачи #40 (02.09)')
    expect(t).toContain('Выполнено: сайт опубликован.')
    expect(t).not.toContain('**')
  })

  it('пустая строка / undefined — не падает', () => {
    expect(mdToPlainText('')).toBe('')
    expect(mdToPlainText(undefined)).toBe('')
  })
})

describe('copyComment', () => {
  let written = []

  beforeEach(() => {
    written = []
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText: vi.fn(async (t) => { written.push(t) }) },
      configurable: true
    })
  })

  it('mode md (default) — дословно, переносы', async () => {
    const body = 'строка 1\n**жирный**\nстрока 3'
    const r = await copyComment(body)
    expect(r.ok).toBe(true)
    expect(written).toEqual([body])
  })

  it('mode text — MD-символы сняты, переносы сохранены', async () => {
    const body = '**жирный** текст\n\n## Заголовок\n- пункт'
    const r = await copyComment(body, 'text')
    expect(r.ok).toBe(true)
    expect(written[0]).toBe('жирный текст\nЗаголовок\n• пункт')
  })

  it('fallback (clipboard API брошен) — execCommand', async () => {
    Object.defineProperty(navigator, 'clipboard', {
      value: { writeText: vi.fn(async () => { throw new Error('denied') }) },
      configurable: true
    })
    const exec = vi.fn(() => true)
    document.execCommand = exec
    const r = await copyComment('plain')
    expect(r.ok).toBe(true)
    expect(r.fallback).toBe(true)
    expect(exec).toHaveBeenCalledWith('copy')
  })
})
