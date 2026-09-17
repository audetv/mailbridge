// v0.25, шаг 7c: утилита расцветки срока (due_date, канон «YYYY-MM-DD»).
import { describe, it, expect } from 'vitest'
import { dueClassOf, todayKey, toCanonicalDate, toDatePickerValue } from '@/utils/due-date'

describe('due-date utils (7c)', () => {
  it('todayKey — календарный день локального TZ в каноне', () => {
    expect(todayKey()).toMatch(/^\d{4}-\d{2}-\d{2}$/)
  })

  it('dueClassOf — по дате: overdue / today / future / none', () => {
    const now = new Date(2026, 8, 17) // 17.09.2026 (локальный TZ)
    expect(dueClassOf({ due_date: '2026-09-16' }, now)).toBe('due--overdue')
    expect(dueClassOf({ due_date: '2026-09-17' }, now)).toBe('due--today')
    expect(dueClassOf({ due_date: '2026-09-18' }, now)).toBe('due--future')
  })

  it('dueClassOf — без срока / мусорный формат → none (не ломается)', () => {
    const now = new Date(2026, 8, 17)
    expect(dueClassOf({}, now)).toBe('due--none')
    expect(dueClassOf({ due_date: null }, now)).toBe('due--none')
    expect(dueClassOf({ due_date: '17.09.2026' }, now)).toBe('due--none')
    expect(dueClassOf({ due_date: '20261340' }, now)).toBe('due--none') // без дефисов — не канон
    expect(dueClassOf(null, now)).toBe('due--none')
  })

  it('toDatePickerValue — канон → Date (локальный TZ), мусор → null', () => {
    const d = toDatePickerValue('2027-01-15')
    expect(d).toBeInstanceOf(Date)
    expect(d.getFullYear()).toBe(2027)
    expect(d.getMonth()).toBe(0)
    expect(d.getDate()).toBe(15)
    expect(toDatePickerValue(null)).toBeNull()
    expect(toDatePickerValue('15-01-2027')).toBeNull()
  })

  it('toCanonicalDate — Date|строка → канон; мусор/пусто → null', () => {
    expect(toCanonicalDate(new Date(2026, 11, 1))).toBe('2026-12-01')
    expect(toCanonicalDate('2027-01-15T00:00:00')).toBe('2027-01-15')
    expect(toCanonicalDate(null)).toBeNull()
    expect(toCanonicalDate(undefined)).toBeNull()
    expect(toCanonicalDate('15.01.2027')).toBeNull()
    // «2026-02-30» нормализуется Date → 02.03 ≠ канон — отклоняем как мусор.
    expect(toCanonicalDate('2026-02-30')).toBeNull()
  })
})
