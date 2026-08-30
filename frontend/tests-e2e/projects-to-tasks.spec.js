// projects-to-tasks.spec.js — e2e-сценарии проекта↔задачи (шаг 13.4)
// Тренd: dev-бекенд :8081 + vite :5173 (прокси /api), авторизация из auth.setup.js.
//
// Сценарии:
//  A. «Проекты → К задачам» → вкладка «Активные» активна + фильтр проекта
//     заполнен + ВСЕ строки таблицы = этот проект.
//  B. Фильтр «Проект» меняет список (разные проекты → разные ID задач).
//  C. Фильтр «Модуль» (epic) в задаче-табах — список = задачи только этого модуля.
//  D. Создание задачи из «Проектов» → задача привязана к проекту (регрессия бага 4),
//     переход на detail, проект закреплён.
//
// Детерминированность: тест A/B/C используют проект «ТРК» и модуль «Сайт ТРК»
// (id=3 — созданы ранее и покрыты seed'ом). Тест D создаёт свой проект+задачу
// через API и не зависит от других тестов (parallel-safe).

import { test, expect } from '@playwright/test'

const BASE = process.env.E2E_BASE_URL || 'http://127.0.0.1:5173'
const PROJECT = 'ТРК'
const EPIC_NAME = 'Сайт ТРК'

// ── Хелперы API ───────────────────────────────────────────────────────────

async function getApi(ctx, path) {
  const r = await ctx.request.get(BASE + path)
  expect(r.ok()).toBe(true)
  return r.json()
}

async function postApi(ctx, path, body) {
  const r = await ctx.request.post(BASE + path, { data: body })
  expect(r.ok(), `POST ${path} → ${r.status()}`).toBe(true)
  return r.json()
}

async function ensureEpic(ctx, projectId) {
  // GET → если модуля нет, создаём. Возвращает {id, name}.
  let list
  try {
    list = await getApi(ctx, `/api/projects/${projectId}/epics`)
  } catch {
    list = []
  }
  const epics = Array.isArray(list) ? list : (list.epics || [])
  let found = epics.find((e) => e.name === EPIC_NAME)
  if (!found) {
    found = await postApi(ctx, `/api/projects/${projectId}/epics`, { name: EPIC_NAME })
  }
  return found
}

async function seedTRK(ctx) {
  // Гарантирует: проект ТРК + модуль Сайт ТРК существуют. Возвращает {project, epic}.
  const projects = await getApi(ctx, '/api/projects')
  const ps = Array.isArray(projects) ? projects : (projects.projects || [])
  let p = ps.find((x) => x.name === PROJECT && !x.archived)
  if (!p) {
    p = await postApi(ctx, '/api/projects', { name: PROJECT, description: 'Торговый комплекс' })
  }
  const epic = await ensureEpic(ctx, p.id)
  return { project: p, epic }
}

// ── Хелперы UI ─────────────────────────────────────────────────────────────

// Индекс колонки по заголовку — слепые индексы td запрещены (порядок колонок
// мог сдвинуться: «Проект» — 7-я колонка, index 6; на index 5 «Приоритет»).
async function colIndex(page, header) {
  const ths = page.locator('thead th')
  const n = await ths.count()
  for (let i = 0; i < n; i++) {
    if ((await ths.nth(i).innerText()).trim() === header) return i
  }
  throw new Error(`колонка «${header}» не найдена: ${(await ths.allInnerTexts()).join('|')}`)
}

async function allRowsProject(page) {
  return columnValues(page, 'Проект')
}

// Значения произвольной колонки TaskTable по заголовку (без слепых индексов td).
async function columnValues(page, header) {
  const idx = await colIndex(page, header)
  const rows = page.locator('tbody tr')
  const n = await rows.count()
  const vals = []
  for (let i = 0; i < n; i++) {
    vals.push((await rows.nth(i).locator('td').nth(idx).innerText()).trim())
  }
  return vals
}

async function goToActive(page) {
  await page.goto('/')
  await expect(page.locator('.tab-bar')).toBeVisible({ timeout: 15000 })
  const activeBtn = page.locator('.tab-bar button', { hasText: 'Активные' })
  if (!(await activeBtn.evaluate((el) => el.classList.contains('active')))) {
    await activeBtn.click()
  }
}

// ══ Сценарий A: Проекты → К задачам ═══════════════════════════════════════

test('A: «К задачам» из Проектов — вкладка Активные + фильтр проекта + все строки = проект', async ({ page }) => {
  const { project: p } = await seedTRK(page.context())

  await page.goto('/#')
  await expect(page.locator('.tab-bar')).toBeVisible({ timeout: 15000 })

  // Переход на вкладку «Проекты»
  await page.locator('.tab-bar button', { hasText: 'Проекты' }).click()
  await expect(page.locator('.projects')).toBeVisible({ timeout: 10000 })

  // Находим строку проекта и жмём «К задачам»
  const row = page.locator('tbody tr', { hasText: p.name }).first()
  await expect(row).toBeVisible({ timeout: 10000 })
  await row.locator('button', { hasText: 'К задачам' }).click()

  // Вкладка «Активные» стала активной
  const activeBtn = page.locator('.tab-bar button', { hasText: 'Активные' })
  await expect
    .poll(async () => activeBtn.evaluate((el) => el.classList.contains('active')), { timeout: 10000 })
    .toBe(true)

  // URL содержит ?tab=active&project=ТРК
  await expect(page).toHaveURL(/tab=active/)
  await expect(page).toHaveURL(/project=/)

  // Селект «Проект» в FilterBar показывает ТРК
  await expectSelected(page, projectSel(page), PROJECT)

  // Ждём данные таблицы
  const tbody = page.locator('tbody tr')
  await expect(tbody.first()).toBeVisible({ timeout: 10000 })

  // ВСЕ строки = этот проект
  const vals = await allRowsProject(page, p.name)
  expect(vals.length).toBeGreaterThan(0)
  for (const v of vals) {
    expect(v).toBe(p.name)
  }
})

// ══ Сценарий B: фильтр «Проект» меняет список ═════════════════════════════

test('B: переключение фильтра «Проект» меняет состав списка задач', async ({ page }) => {
  await seedTRK(page.context())
  // Второй проект для контраста
  let pOther
  const projects = await getApi(page.context(), '/api/projects')
  const ps = Array.isArray(projects) ? projects : (projects.projects || [])
  pOther = ps.find((x) => x.name !== PROJECT && x.name !== 'Входящие' && !x.archived)
  if (!pOther) {
    pOther = await postApi(page.context(), '/api/projects', { name: 'Контраст E2E', description: 'для теста' })
  }

  await goToActive(page)
  await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 })

  const firstCol = async () => {
    const rows = page.locator('tbody tr')
    const n = await rows.count()
    const ids = []
    for (let i = 0; i < n; i++) ids.push((await rows.nth(i).locator('td').first().innerText()).trim())
    return ids
  }

  const base = await firstCol()
  expect(base.length).toBeGreaterThan(0)

  // Препод-условие: в общем списке есть задачи НЕ ТРК — иначе «список изменился»
  // нельзя было бы наблюдать.
  expect(
    (await allRowsProject(page)).some((v) => v !== PROJECT),
    'в dev-БД нет задач других проектов — сценарий B невалиден'
  ).toBe(true)

  // Включаем фильтр ТРК и проверяем, что выбор прижился (лейбл селекта)
  await openSelect(page)
  await pickOption(page, PROJECT)
  await expectSelected(page, projectSel(page), PROJECT)

  // Ждём, пока таблица ПО-НАСТОЯЩЕМУ отфильтрована: все видимые строки = ТРК.
  // (Слабая проверка «хотя бы одна строка ТРК» недостаточна: она проходит и
  // на неотфильтрованной таблице.)
  await expect
    .poll(async () => {
      const vals = await allRowsProject(page)
      return vals.length > 0 && vals.every((v) => v === PROJECT)
    }, { timeout: 15000, message: 'все строки должны быть проекта ТРК' })
    .toBe(true)

  const trkIDs = await firstCol()
  expect(trkIDs.length).toBeGreaterThan(0)
  expect(trkIDs).not.toEqual(base) // список изменился (у ТРК меньше задач)

  // Переключаем на другой проект — список снова меняется
  await openSelect(page)
  await pickOption(page, pOther.name)
  await expectSelected(page, projectSel(page), pOther.name)

  // Пустая таблица допустима (у pOther может не быть задач): ждём, что либо
  // нет строк, либо все видимые = pOther.
  await expect
    .poll(async () => {
      const vals = await allRowsProject(page)
      return vals.length === 0 || vals.every((v) => v === pOther.name)
    }, { timeout: 15000, message: 'все строки должны быть проекта ' + pOther.name })
    .toBe(true)

  const otherIDs = await firstCol()
  expect(otherIDs).not.toEqual(trkIDs)
  if (otherIDs.length > 0) {
    const vals2 = await allRowsProject(page)
    for (const v of vals2) expect(v).toBe(pOther.name)
  }
})

// ══ Сценарий C: фильтр «Модуль» ═══════════════════════════════════════════

test('C: фильтр «Модуль» — список = только задачи этого модуля', async ({ page }) => {
  const { epic } = await seedTRK(page.context())

  await goToActive(page)
  // Сначала ставим фильтр проекта (у модулей опции подгружаются по проекту)
  await openSelect(page)
  await pickOption(page, PROJECT)
  await expectSelected(page, projectSel(page), PROJECT)
  await expect(page.locator('tbody tr').first()).toBeVisible({ timeout: 10000 })

  // Ждём, что у ТРК есть хотя бы одна задача
  const taskRows = page.locator('tbody tr')
  const before = await taskRows.count()
  expect(before).toBeGreaterThan(0)

  // Открываем селект «Модуль»
  await openSelect(page, 'epic')
  // Опции модулей подгружаются асинхронно — pickOption сам ждёт опцию
  const val = await pickOptionExpect(page, epic.name)
  expect(val.endsWith(epic.name), `опция «${val}» должна заканчиваться «${epic.name}»`).toBe(true)
  await expectSelected(page, epicSel(page), epic.name)

  // Таблица ПО-НАСТОЯЩЕМУ отфильтрована по модулю: либо пустая (у модуля нет
  // задач), либо все видимые строки имеют этот модуль.
  await expect
    .poll(async () => {
      const vals = await columnValues(page, 'Модуль')
      return vals.length === 0 || vals.every((v) => v.includes(epic.name))
    }, { timeout: 15000, message: 'все строки должны быть модуля «' + epic.name + '»' })
    .toBe(true)
})

// ══ Сценарий D: создание из Проектов → задача привязана (регрессия бага 4) ═

test('D: «Создать» из Проектов → задача с привязкой к проекту', async ({ page }) => {
  const ts = Date.now().toString().slice(-6)
  const uniqueName = `E2E Д${ts}`

  // Создаём уникальный проект через API (параллель-безопасно)
  await postApi(page.context(), '/api/projects', { name: uniqueName, description: 'temp e2e' })

  await page.goto('/#')
  await expect(page.locator('.tab-bar')).toBeVisible({ timeout: 15000 })
  await page.locator('.tab-bar button', { hasText: 'Проекты' }).click()
  await expect(page.locator('.projects')).toBeVisible({ timeout: 10000 })

  // Ждём появления нашего проекта в таблице
  const row = page.locator('tbody tr', { hasText: uniqueName }).first()
  await expect(row).toBeVisible({ timeout: 10000 })

  // Кнопка «Создать» в колонке «Задача»
  await row.locator('button', { hasText: 'Создать' }).click()

  // Диалог «Новая задача» — проект закреплён (lock-project=true)
  const dialog = page.locator('.p-dialog, [role="dialog"]').last()
  await expect(dialog).toBeVisible({ timeout: 5000 })
  await expect(dialog).toContainText(uniqueName) // проект показан как контекст

  // Заполняем заголовок
  const titleInput = dialog.locator('input[type="text"], input:not([type])').first()
  await titleInput.fill(`Задача ${ts}`)

  // Кнопка «Создать»
  await dialog.locator('button', { hasText: 'Создать' }).click()

  // Переход на /tasks/:id
  await expect(page).toHaveURL(/\/tasks\/\d+/, { timeout: 10000 })

  // Проверяем привязку через API
  const id = page.url().match(/\/tasks\/(\d+)/)?.[1]
  const task = await (await page.context().request.get(BASE + `/api/tasks/${id}`)).json()
  const t = task.task || task
  expect(t.project).toBe(uniqueName)
  expect(t.status).toBe('new')
})

// ── Вспомогательные UI-хелперы (PrimeVue Select в FilterBar) ──────────────
//
// Структура FilterBar.vue: [InputText.search-input, Select «Проект»,
// Select «Модуль» (class .epic-select)]. Селекторы — ПОЗИЦИОННЫЕ, не
// hasText: после выбора значения текст плейсхолдера («Проект»/«Модуль»)
// ВЕРНУЕТСЯ значением, и `.filter({hasText:'Проект'})` больше не находит
// селект (видели на B: выбор ТРК прошёл, a expectSelected не находил).
//
// Прочие нюансы (проверены отладочными прогонями 2026-08-30):
//  1. Клик по корневому .p-select контейнеру НЕ открывает панель — filter-bar
//     перехватывает pointer events по центру над опциями. Нужен клик по
//     span[role=combobox].
//  2. Нативный el.click() НЕ выбирает опцию — Listbox слушает mousedown.
//  3. Playwright locator.click() на опции падает actionability-race
//     («element detached during click») — панель ре-рендерится
//     (data-p-focused мигрирует). Надёжный приём: нативный mousedown+
//     mouseup+click через evaluate.
//  4. Панель опций — единственный [role=listbox]/[role=option]-контейнер в
//     документe в любой момент времени (второй селект закрыт).

function projectSel(page) {
  return page.locator('.filter-bar .p-select:not(.epic-select)').first()
}
function epicSel(page) {
  return page.locator('.filter-bar .p-select.epic-select').first()
}

// Нативный mousedown+mouseup+click: PrimeVue Listbox слушает mousedown,
// а ре-рендер панели ломает locator.click() (element detached).
async function nativeClickOption(page, text) {
  const opt = page.locator('.p-select-option, [role="option"]', { hasText: text }).first()
  await opt.evaluate((el) => {
    const r = el.getBoundingClientRect()
    const x = r.left + r.width / 2
    const y = r.top + r.height / 2
    for (const type of ['mousedown', 'mouseup', 'click']) {
      el.dispatchEvent(new MouseEvent(type, { bubbles: true, cancelable: true, view: window, clientX: x, clientY: y }))
    }
  })
}

// Открывает дропдаун (по умолчанию проект; where='epic' — модуль).
async function openSelect(page, where = 'project') {
  const sel = where === 'epic' ? epicSel(page) : projectSel(page)
  const btn = sel.locator('[role="combobox"]').first()
  const panel = page.locator('[role="listbox"]').first()
  if (await panel.isVisible().catch(() => false)) {
    await page.mouse.click(10, 10) // клик мимо закрывает чужую панель
    await page.waitForTimeout(150)
  }
  await btn.click()
  await expect(page.locator('[role="option"]').first()).toBeVisible({ timeout: 5000 })
}

// Выбирает опцию в открытом дропдауне (см. openSelect): нативная мышь (см.
// nativeClickOption) — PrimeVue слушает mousedown, а locator.click() ловит
// re-render race.
async function pickOption(page, text) {
  const opt = page.locator('.p-select-option, [role="option"]', { hasText: text }).first()
  await expect(opt).toBeVisible({ timeout: 5000 }, 'опция «' + text + '» не найдена в выпадающем списке')
  await nativeClickOption(page, text)
}

// Как pickOption, но возвращает aria-label опции (проверка, что выбран
// именно нужный модуль, а не похожий по тексту).
async function pickOptionExpect(page, text) {
  const opt = page.locator('.p-select-option, [role="option"]', { hasText: text }).first()
  await expect(opt).toBeVisible({ timeout: 5000 }, 'опция «' + text + '» не найдена в выпадающем списке')
  const val = await opt.getAttribute('aria-label')
  await nativeClickOption(page, text)
  return val
}

// Пост-проверка: селект (projectSel/epicSel) показывает выбранный текст.
async function expectSelected(page, sel, value) {
  await expect
    .poll(async () => (await sel.innerText()).trim(), {
      timeout: 8000,
      message: 'селект должен показывать «' + value + '»',
    })
    .toContain(value)
}
