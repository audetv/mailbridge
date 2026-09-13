// copy-comment.js — копирование комментария/ответа в буфер в режиме
// «чистый текст» (по умолчанию) или «как есть (MD)».
//
// Тела комментариев у нас — plain text, иногда с фрагментами MD
// (жирный **x**, заголовки ##, списки -). UI рендерит их дословно,
// поэтому «как MD» = дословно, с переносами.
//
// «Как TXT» снимает MD-символы для вставки в Outlook/Telegram/почту,
// где **x** покажется как звёздочки.

/** Убирает MD-разметку: **жирный** → жирный, `код` → код, #х → х,
 *  [текст](url) → текст, > цитата → цитата, - пункт → • пункт.
 *  Не ломает обычные переносы.
 * @param {string} md
 * @returns {string}
 */
export function mdToPlainText(md) {
  if (typeof md !== 'string' || md.length === 0) return ''
  let t = md
  // ссылки: [text](url) → text (списки/код-блоки не трогаем)
  t = t.replace(/\[([^\]]+)\]\([^)]*\)/g, '$1')
  // заголовки: # / ## / ### → просто текст
  t = t.replace(/^\s*#{1,6}\s+/gm, '')
  // жирный/курсив: **x**, __x__, *x*, _x_ → x
  t = t.replace(/(\*\*|__)([^*_]+)\1/g, '$2')
  t = t.replace(/(^|[\s(])\*([^*_\s][^*]*)\*(?=[\s),.!?]|$)/gm, '$1$2')
  t = t.replace(/(^|[\s(])_([^_\s][^_]*)_(?=[\s),.!?]|$)/gm, '$1$2')
  // инлайн-код: `код` → код
  t = t.replace(/`([^`]+)`/g, '$1')
  // строгий (~~x~~), если встречается
  t = t.replace(/~~([^~]+)~~/g, '$1')
  // цитата: > текст → текст
  t = t.replace(/^\s*>\s?/gm, '')
  // списки: -, *, +, 1. → • пункт
  t = t.replace(/^(\s*)[-*+]\s+/gm, '$1• ')
  t = t.replace(/^(\s*)\d+\.\s+/gm, '$1• ')
  return t.replace(/\n{3,}/g, '\n\n').trimEnd()
}

/** Копирует в clipboard: «md» (по умолчанию) или «text».
 *  Возвращает {ok, text, fallback}.
 * @param {string} body
 * @param {'md'|'text'} mode
 * @returns {Promise<{ok:boolean,text:string,fallback:boolean}>}
 */
export async function copyComment(body, mode = 'md') {
  const text = mode === 'text' ? mdToPlainText(body) : (body || '')
  // Clipboard API — современный Chrome/Safari/Firefox (https/localhost)
  try {
    await navigator.clipboard.writeText(text)
    return { ok: true, text, fallback: false }
  } catch {
    // Fallback: legacy textarea+execCommand (браузеры без clipboard API
    // или http-локальные)
    const ta = document.createElement('textarea')
    ta.value = text
    ta.style.position = 'fixed'
    ta.style.opacity = '0'
    ta.style.pointerEvents = 'none'
    document.body.appendChild(ta)
    let ok = false
    try {
      ta.select()
      ok = document.execCommand('copy')
    } finally {
      document.body.removeChild(ta)
    }
    return { ok, text, fallback: true }
  }
}
