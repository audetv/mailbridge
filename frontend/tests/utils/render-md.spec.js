// @vitest-environment jsdom
// jsdom (не happy-dom): в happy-dom + DOMPurify вырезается первый элемент
// фрагмента (h1/ul/ol). jsdom — штатное поведение; браузер — штатное.
import { describe, it, expect } from 'vitest'
import { renderMd, looksLikeMd } from '@/utils/render-md'

const textOf = (frag) => frag.replace(/<[^>]+>/g, '')

describe('renderMd (v0.23, step 3b)', () => {
  it('empty => ""', () => {
    expect(renderMd('')).toBe('')
    expect(renderMd(null)).toBe('')
  })

  it('plain text stays', () => {
    const out = textOf(renderMd('Dobryy den, eto test.'))
    expect(out).toContain('Dobryy den, eto test.')
  })

  it('headings ##/### render', () => {
    const out = renderMd('## Razdel\n\n### Pod\n')
    expect(out).toMatch(/<h2[^>]*>Razdel<\/h2>/)
    expect(out).toMatch(/<h3[^>]*>Pod<\/h3>/)
  })

  it('bold **x** renders strong', () => {
    expect(renderMd('eto **zhirnyy** tekst')).toMatch(/<strong>zhirnyy<\/strong>/)
  })

  it('italic *x* renders em', () => {
    expect(renderMd('eto *kurziv* tekst')).toMatch(/<em>kurziv<\/em>/)
  })

  it('ul/ol lists render', () => {
    const a = renderMd('- one\n- two')
    expect(a).toMatch(/<ul>/)
    expect(a).toMatch(/<li>one<\/li>/)
    const b = renderMd('1. one\n2. two')
    expect(b).toMatch(/<ol>/)
    expect(b).toMatch(/<li>one<\/li>/)
  })

  it('task list checkbox renders', () => {
    const out = renderMd('- [ ] open\n- [x] done')
    expect(out).toMatch(/<input[^>]*type="checkbox"/)
    expect(out).toMatch(/checked/i)
  })

  it('inline code + fence render', () => {
    expect(renderMd('run `npm test` now')).toMatch(/<code>npm? test<\/code>|<code>npm test<\/code>/)
    const f = renderMd('```\nfunc main() {}\n```')
    expect(f).toMatch(/<pre>/)
    expect(f).toMatch(/<code>/)
    expect(f).toContain('func main()')
  })

  it('hr and blockquote render', () => {
    expect(renderMd('a\n\n---\n\nb')).toMatch(/<hr/)
    expect(renderMd('> citata')).toMatch(/<blockquote>/)
  })

  it('md link gets target=_blank rel=noopener http(s) only', () => {
    const out = renderMd('[doc](https://a.io/d)')
    expect(out).toMatch(/<a[^>]*href="https:\/\/a.io\/d"/)
    expect(out).toMatch(/target="_blank"/)
    expect(out).toMatch(/rel="noopener noreferrer"|rel="[^"]*noopener/)
  })

  it('bare URL auto-links in new tab, trailing comma stays out', () => {
    const out = renderMd('look at https://kushniras.ru/tz/2025/ then reply.')
    const m = out.match(/<a[^>]*href="([^"]+)"/)
    expect(m).toBeTruthy()
    expect(m[1]).toBe('https://kushniras.ru/tz/2025/')
    expect(out).toMatch(/target="_blank"/)
    expect(out).toMatch(/noopener/)
    // «reply.» text intact
    expect(textOf(out)).toContain('reply.')
  })

  it('www. — линкуется (marked автолинкует сам)', () => {
    const out = renderMd('site: www.example.com ok')
    expect(out).toMatch(/<a[^>]*href="(http|https):\/\/www\.example\.com"/)
  })

  it('SCRIPT tag & onerror handlers are removed', () => {
    const tag = '<scr' + 'ipt>alert(1)</' + 'script>'
    const out = renderMd('ok ' + tag + '\n\n[x](1)')
    expect(out).not.toMatch(/<script/i)
    expect(out).not.toContain('alert(1)')
    expect(out).not.toMatch(/<img/i)
  })

  it('javascript:/data: link hrefs are stripped', () => {
    expect(renderMd('[x](javascript:alert(1))')).not.toMatch(/href="javascript:/i)
    expect(renderMd('[x](data:text/html;base64,AA)')).not.toMatch(/href="data:/i)
  })
})

describe('looksLikeMd — детектор MD (plain vs MD)', () => {
  it('обычный текст без MD → false', () => {
    expect(looksLikeMd('Ссылка на ТЗ, о котором идет речь: https://kushniras.ru/seo/x.html')).toBe(false)
    expect(looksLikeMd('Просто текст без MD. И без списков.')).toBe(false)
    expect(looksLikeMd('Добрый день.')).toBe(false)
  })

  it('markdown-синтаксис → true', () => {
    expect(looksLikeMd('## Раздел\n\nтекст')).toBe(true)
    expect(looksLikeMd('это **жирный** текст')).toBe(true)
    expect(looksLikeMd('- пункт 1\n- пункт 2')).toBe(true)
    expect(looksLikeMd('- [ ] open\n- [x] done')).toBe(true)
    expect(looksLikeMd('1. one\n2. two')).toBe(true)
    expect(looksLikeMd('`inline code` here')).toBe(true)
    expect(looksLikeMd('```\ncode fence\n```')).toBe(true)
    expect(looksLikeMd('> цитата')).toBe(true)
    expect(looksLikeMd('---\nhorizontal rule')).toBe(true)
  })

  it('empty/undefined → false', () => {
    expect(looksLikeMd('')).toBe(false)
    expect(looksLikeMd(null)).toBe(false)
    expect(looksLikeMd(undefined)).toBe(false)
  })
})

  it('bare URL с завершающей точкой → точка остаётся в тексте, href чистый', () => {
    const out = renderMd('См. https://max.ru/channel; это чат.')
    expect(out).toMatch(/<a[^>]*href="https:\/\/max\.ru\/channel"[^>]*>https:\/\/max\.ru\/channel<\/a>;/)
  })
