// Срок задачи (due_date, канон «YYYY-MM-DD» — онтология v0.5.2 §7.5).
// Расцветка утверждённого срока (7c): просрочено / сегодня / дальше / без срока.
// Сравнение по КАЛЕНДАРНЫМ дням (UTC-независимо: дата без времени).

export const DUE_CLASS_OVERDUE = 'due--overdue'
export const DUE_CLASS_TODAY = 'due--today'
export const DUE_CLASS_FUTURE = 'due--future'
export const DUE_CLASS_NONE = 'due--none'

function dateKey(d) {
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(
    d.getDate()
  ).padStart(2, '0')}`
}

/**
 * dueClassOf(task) → css-класс расцветки по `due_date` задачи.
 * 'YYYY-MM-DD' < сегодня → overdue; = сегодня → today; > → future; нет → none.
 */
export function dueClassOf(task, now = new Date()) {
  const due = task?.due_date
  if (!due) return DUE_CLASS_NONE
  if (typeof due !== 'string' || !/^\d{4}-\d{2}-\d{2}$/.test(due)) return DUE_CLASS_NONE
  return due < dateKey(now) ? DUE_CLASS_OVERDUE : due === dateKey(now) ? DUE_CLASS_TODAY : DUE_CLASS_FUTURE
}

/** todayKey(now) → «YYYY-MM-DD» календарного дня (локальный TZ). */
export function todayKey(now = new Date()) {
  return dateKey(now)
}

/**
 * toISODateValue(d) → значение для PrimeVue DatePicker (Date) из канона «YYYY-MM-DD».
 * null/мусор → null.
 */
export function toDatePickerValue(due) {
  if (!due || typeof due !== 'string') return null
  const m = due.match(/^(\d{4})-(\d{2})-(\d{2})/)
  if (!m) return null
  const d = new Date(Number(m[1]), Number(m[2]) - 1, Number(m[3]))
  return Number.isNaN(d.getTime()) ? null : d
}

/**
 * toCanonicalDate(d) → канон «YYYY-MM-DD» из Date (DatePicker) | строки даты.
 * Мусор/не-дата → null.
 */
export function toCanonicalDate(value) {
  if (value == null) return null
  if (value instanceof Date && !Number.isNaN(value.getTime())) return dateKey(value)
  if (typeof value === 'string' && /^\d{4}-\d{2}-\d{2}/.test(value)) {
    const [y, m, d] = value.slice(0, 10).split('-').map(Number)
    const dt = new Date(y, m - 1, d)
    if (!Number.isNaN(dt.getTime()) && dateKey(dt) === value.slice(0, 10)) return value.slice(0, 10)
  }
  return null
}
