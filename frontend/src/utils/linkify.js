// linkify.js — автолинкинг ссылок в plain- тексте комментария.
// Делим тело на сегменты {text, href|null}: текст — как есть (text-ноды,
// XSS-чисто, v-html НЕ используем), URL — <a target="_blank" rel="noopener">.
// Обычное поведение почты/Slack/GitHub: голые https:// ссылки кликаются
// и открываются в новой вкладке. Ссылки в MD-форме [текст](url) у нас
// в данных не встречаются — не обрабатываем.

const URL_RE = /(https?:\/\/[^\s<>"']+)/g
// Точки/запятые/точки-с-запятой/скобки, «зависшие» в конце URL
// (встречаются: "https://max.ru/channel; Tel" и т.п.) — не часть ссылки.
const TRAILING_PUNCT = /[.,;:!?'”“)"»]+$/

/** @param {string} body текст комментария
 * @returns {Array<{text:string, href:string|null}>} */
export function linkify(body) {
  const out = []
  if (typeof body !== 'string' || body.length === 0) return out
  URL_RE.lastIndex = 0
  let last = 0
  let m
  while ((m = URL_RE.exec(body)) !== null) {
    if (m.index > last) out.push({ text: body.slice(last, m.index), href: null })
    const raw = m[1]
    const href = raw.replace(TRAILING_PUNCT, '')
    out.push({ text: raw, href })
    last = URL_RE.lastIndex
  }
  if (last < body.length) out.push({ text: body.slice(last), href: null })
  return out
}
