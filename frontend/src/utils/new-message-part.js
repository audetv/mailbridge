// v0.23, шаг 5 (решение владельца, 14.09, задача 378 / письма Маргариты #473/#474):
// некоторые почтовые клиенты (настройка «сохранять историю переписки») физически
// вставляют ВСЮ историю внутрь ТЕКСТА каждого нового письма: тело =
//   [НОВОЕ] + [старая история целиком, включая подпись и «Кому:»]
// (проверено по БД: #474 (24631 зн) = 972 зн нового + полное тело #473 (23659 зн)).
//
// Поэтому «оригинал письма» показываем только как ЕГО НОВУЮ ЧАСТЬ:
//   1) diff: если тело = [новое] + [полное предыдущее письмо треда] — отбрасываем
//      хвост-историю (новое всегда сверху, у такого типа писем);
//   2) отсечение мусора: режем по первому маркеру подписи/шапки истории
//      («-- Просьба…», «С уважением…», «Кому:», «From: … [mailto:», «Тема:»,
//      маркер «DD.MM.YYYY, HH:MM»);
//   3) fallback без нового (пусто) → хвост письма (последние NEW_PART_TAIL зн).
// Полное письмо доступно: «Показать полностью» (карточка) и /inbox/{id} (лента).
//
// ВАЖНО: в body_text у таких писем НЕТ переноса строк (HTML→text без <br>) —
// все маркеры ищутся на уровне символов, \s* обязательное.

// Маркеры «конец нового, дальше мусор»: подписи, шапки «Кому/Тема», Outlook-From,
// маркеры тредa outlook-типа. Первый найденный = отсечение.
const NOISE = /\s*(?:--\s+\S|С уважением|С благодарностью|С наилучшими|С пожеланиями|Best regards|Kind regards|Кому:|Тема:|From:\s*\S.*\[mailto:|Sent:|Subject:)|\s*\d{1,2}\.\d{1,2}\.\d{4},\s*\d{1,2}:\d{2}/

const MAX_NEW_PART = 8000 // sanity: «новое» больше — хвост, см. ниже
const NEW_PART_TAIL = 800 // fallback (нет нового): сколько знаков с конца показать

function firstNoiseCut(text) {
  const m = NOISE.exec(text)
  return m ? m.index : -1
}

/**
 * Новая часть письма (то, что написали в этом сообщении — не история и не подпись).
 * @param {string} body   тело письма (inbox_items.body_text)
 * @param {string} [prevBody] тело предыдущего письма в трéде ('' если нет)
 * @returns {string}
 */
export function newPartOf(body, prevBody = '') {
  const text = (body || '').trim()
  if (!text) return ''

  // 1) diff: тело = [новое] + [полное прежнее письмо] (проверено: indexOf сверху)
  let x = text
  if (prevBody && text.length > prevBody.length) {
    const idx = text.indexOf(prevBody)
    if (idx === 0) {
      // «нового» нет вообще — тело = прежнее письмо целиком (форвард). Показываем хвост
      return text.length > NEW_PART_TAIL ? '…' + text.slice(-NEW_PART_TAIL) : text
    }
    if (idx > 0) {
      x = text.slice(0, idx)
    }
  }
  // 2) отсечь мусор по первому маркеру подписи/шапки
  const cut = firstNoiseCut(x)
  const s = (cut === -1 ? x : x.slice(0, cut)).trim()

  // 3) fallback: нового нет (письмо = только история/подпись) → хвост письма
  if (!s) {
    return text.length > NEW_PART_TAIL ? '…' + text.slice(-NEW_PART_TAIL) : text
  }
  if (s.length > MAX_NEW_PART) {
    return '…' + s.slice(-MAX_NEW_PART)
  }
  return s
}

// Публичное имя (используется TaskDetailView как preview).
export function newMessagePartOf(body, prevBody = '') {
  return newPartOf(body, prevBody)
}

// Нормализация whitespace для сравнения «совпадает ли письмо с предыдущим».
export function normalizeText(s) {
  return (s || '').replace(/\s+/g, ' ').trim()
}
