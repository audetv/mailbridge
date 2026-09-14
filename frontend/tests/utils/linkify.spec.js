import { describe, it, expect } from 'vitest'
import { linkify } from '@/utils/linkify'

describe('linkify — автолинкинг plain-текста', () => {
  it('текст без ссылок — один текстовый сегмент, href=null', () => {
    const segs = linkify('Просто текст без ссылок')
    expect(segs).toEqual([{ text: 'Просто текст без ссылок', href: null }])
  })

  it('голый https URL — сегмент с href', () => {
    const segs = linkify('Детали здесь: https://example.com/page?x=1')
    expect(segs.some(s => s.href === 'https://example.com/page?x=1')).toBe(true)
    expect(segs.some(s => s.text === 'Детали здесь: ')).toBe(true)
  })

  it('http (без s) тоже линкуется', () => {
    const segs = linkify('http://example.org')
    expect(segs[0].href).toBe('http://example.org')
  })

  it('несколько ссылок — все сегментированы', () => {
    const segs = linkify('a https://a.io/1 b https://b.io/2 c')
    const hrefs = segs.filter(s => s.href !== null).map(s => s.href)
    expect(hrefs).toEqual(['https://a.io/1', 'https://b.io/2'])
  })

  it('висячая точка/запятая/точка-с-запятой в конце — не часть href (текст-сегмент хранит полное)', () => {
    const segs = linkify('канал: https://max.ru/channel_musa; tel 200')
    const link = segs.find(s => s.href)
    expect(link.href).toBe('https://max.ru/channel_musa')
    expect(link.text).toBe('https://max.ru/channel_musa;')
  })

  it('обрезает висящие кавычки/скобки: "https://a.io/1)"', () => {
    const segs = linkify('ссылка "https://a.io/1"' + ')')
    const link = segs.find(s => s.href)
    expect(link.href).toBe('https://a.io/1')
  })

  it('пустая строка — без сегментов', () => {
    expect(linkify('')).toEqual([])
    expect(linkify(null)).toEqual([])
  })

  it('URL в середине строки с кириллицей вокруг — работает', () => {
    const segs = linkify('См. https://kushniras.ru/seo/lider-sport.ru/ и дальше текст')
    expect(segs.find(s => s.href)?.href).toBe('https://kushniras.ru/seo/lider-sport.ru/')
  })

  it('MD-ссылка [текст](url) — «[текст]» остаётся текстом, URL внутри линкуется без скобок', () => {
    const segs = linkify('[текст](https://a.io)')
    expect(segs.some(s => s.text.includes('[текст]'))).toBe(true)
    expect(segs.find(s => s.href)?.href).toBe('https://a.io')
  })
})
