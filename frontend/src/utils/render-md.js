// render-md.js — безопасный рендер Markdown комментариев (v0.23, шаг 3b).
//
// Стек: marked (GFM: заголовки, **жирный**, списки, task-lists, `код`,
// ```, ---, >, [текст](url) + автолинкинг голых URL с корректным
// отделением хвостовой пунктуации) → DOMPurify (allowlist + хуки) →
// все <a> получают target="_blank" rel="noopener noreferrer".
//
// Безопасность (это v-html — критично):
//   - DOMPurify allowlist: <script>, on*, javascript:/data: — не проходят;
//   - afterSanitizeAttributes: у <a> только http(s)/mailto +
//     target="_blank" rel="noopener noreferrer";
//   - только безопасные <input type=checkbox disabled> (GFM task-lists),
//     любое другое <input> удаляется;
//   - <img> — src только http(s)/относительный.
//
// UI: CommentList рендерит:
//   <div class="comment-body md" v-html="renderMd(comment.body)">
//   или (plain) прежний текст-рендер linkify() — без v-html.

import { marked } from 'marked'
import DOMPurify from 'dompurify'

marked.setOptions({
  gfm: true,    // task-lists, strikethrough, автолинкинг голых URL
  breaks: true, // жёсткие переносы строк — как у pre-wrap в plain-рендере
})

const PURIFY_CONFIG = {
  ALLOWED_TAGS: [
    'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
    'p', 'br', 'hr',
    'strong', 'b', 'em', 'i', 'del', 's', 'u', 'mark', 'code', 'pre',
    'blockquote',
    'ul', 'ol', 'li',
    'table', 'thead', 'tbody', 'tr', 'th', 'td',
    'a', 'img',
    'input', // Только GFM checkbox, хук фильтрует
  ],
  ALLOWED_ATTR: [
    'href', 'title', 'target', 'rel',
    'src', 'alt', 'width', 'height',
    'start', 'type', 'checked', 'disabled', 'value',
    'colspan', 'rowspan', 'scope',
  ],
  ALLOW_DATA_ATTR: false,
}

const purify = DOMPurify(window)
purify.addHook('afterSanitizeAttributes', (node) => {
  // on* — параноидально ещё раз
  for (const a of [...node.attributes]) {
    if (/^on/i.test(a.name)) node.removeAttribute(a.name)
  }
  if (node.tagName === 'A') {
    const href = (node.getAttribute('href') || '').trim()
    if (!/^(https?|mailto):/i.test(href)) node.removeAttribute('href')
    node.setAttribute('target', '_blank')
    node.setAttribute('rel', 'noopener noreferrer')
  }
  if (node.tagName === 'INPUT') {
    if ((node.getAttribute('type') || '').toLowerCase() !== 'checkbox') {
      return false // DOMPurify сам удалит узел (безопасно во время обхода)
    }
    node.setAttribute('type', 'checkbox')
    node.setAttribute('disabled', 'disabled')
    for (const a of ['name', 'value', 'formaction', 'action', 'form', 'id', 'onchange']) {
      node.removeAttribute(a)
    }
  }
  if (node.tagName === 'IMG') {
    const src = (node.getAttribute('src') || '').trim()
    if (!/^(https?:)?\//i.test(src)) node.setAttribute('src', '')
  }
})

function escapeHtml(s) {
  return s.replace(/[&<>"']/g, (c) => ({
    '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;', "'": '&#39;'
  }[c]))
}

// --- Публичные API ---

/** Рендерит markdown в безопасный HTML (для v-html). */
export function renderMd(body) {
  if (typeof body !== 'string' || body.length === 0) return ''
  let raw
  try {
    raw = marked.parse(body) || ''
  } catch (e) {
    console.warn('renderMd: parse failed', e)
    raw = escapeHtml(body)
  }
  const clean = purify.sanitize(raw, PURIFY_CONFIG)
  return typeof clean === 'string' ? clean : String(clean)
}

/**
 * Детектор: содержит ли текст синтаксис Markdown (в наших данных он
 * появляется только в ответах/отчётах с явным MD).
 * Возвращает true, если есть хотя бы один явный MD-конструкции на строке.
 */
export function looksLikeMd(body) {
  if (typeof body !== 'string') return false
  const lines = body.split('\n')
  const strongInline = /\*\*[^*\n]+\*\*/
  const italicSingle = /(^|\s)\*[^*\n]+\*(?!\*)/
  const heading = /^\s{0,3}#{1,6}\s+\S/
  const ulItem = /^\s{0,3}[-*+]\s+\S/
  const numItem = /^\s{0,3}\d+\.\s+\S/
  const blockquote = /^\s{0,3}>\s?\S/
  const hr = /^\s{0,3}(---+|\*\*\*+|___+)\s*$/
  const fence = /^\s{0,3}```/
  const inlineCode = /(^|\s)`[^`\n]+`(?=\s|$)/
  const mdLink = /\[[^]\n]+\]\((https?:|mailto:)[^)\s]+\)/
  for (const line of lines) {
    if (strongInline.test(line)) return true
    if (italicSingle.test(line)) return true
    if (heading.test(line)) return true
    if (ulItem.test(line)) return true
    if (numItem.test(line)) return true
    if (blockquote.test(line)) return true
    if (hr.test(line)) return true
    if (fence.test(line)) return true
    if (inlineCode.test(line)) return true
    if (mdLink.test(line)) return true
  }
  return false
}

export default { renderMd, looksLikeMd }
