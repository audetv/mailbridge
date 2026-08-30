// seed-dev.mjs — идемпотентный seed dev-БД для e2e (шаги 13.3–13.4)
// Поднимает бекенд dev (8081) и гарантирует фиксированные ФИКСАЦИИ:
//   1. Проекты (ТРК, Отель, Лидер Спорт, Театр + «Входящие»-fallback).
//   2. Модуль «Сайт ТРК» в проекте ТРК.
//   3. Минимум одна задача в ТРК и в модуле «Сайт ТРК» (прек-условия A/B/C).
//   4. Минимум одна задача НЕ в ТРК (прек-условие B: «список изменился»).
// Идемпотентность: всё создаём только если ещё нет (проверка по API).
// Пользователь периодически копирует прод-БД в dev → seed обязан сам
// восстанавливать фитуры, не полагаясь на текущее состояние БД.
// Запуск: cd frontend && npm run e2e:seed  (или node scripts/seed-dev.mjs [BASE_URL])
//   BASE_URL — http://127.0.0.1:5173 (vite-прокси) или http://127.0.0.1:8081

const BASE = process.argv[2] || process.env.E2E_BASE_URL || 'http://127.0.0.1:5173'

const PROJECTS = [
  { name: 'Входящие', description: 'Fallback-проект' },
  { name: 'ТРК', description: 'Торговый комплекс' },
  { name: 'Отель', description: 'Отель/гостиница' },
  { name: 'Лидер Спорт', description: 'Фитнес' },
  { name: 'Театр', description: 'Театр' }
]

// Фиксированные задачи (e2e-фитуры; заголовок помечен E2E).
const TASK_FIXTURES = [
  { project: 'ТРК', epic: 'Сайт ТРК', title: 'E2E: задача в модуле Сайт ТРК — обслуживание торгового комплекса' },
  { project: 'Отель', epic: null, title: 'E2E: задача Отель — бронирование номера' }
]

function fail(err) {
  console.error('seed-dev ERROR:', err.message)
  process.exit(1)
}

async function api(path, opts = {}) {
  let res
  try {
    res = await fetch(BASE + path, {
      ...opts,
      headers: { 'Content-Type': 'application/json', ...(opts.headers || {}) }
    })
  } catch (e) {
    fail(new Error(`fetch ${path}: ${e.message} — бекенд ${BASE} не отвечает? Поднимите: make run-dev + npm run dev`))
  }
  const ct = res.headers.get('content-type') || ''
  const isJson = ct.includes('application/json')
  const body = isJson ? await res.json().catch(() => ({})) : await res.text().catch(() => '')
  return { status: res.status, body }
}

async function login() {
  const r = await api('/api/auth/login', {
    method: 'POST',
    body: JSON.stringify({ username: 'admin', password: 'admin' })
  })
  if (r.status !== 200) {
    fail(new Error(`login failed: ${r.status} (ожидается admin/admin на dev :8081; см. configs/config.env)`))
  }
  return r.body.token
}

const auth = (token) => ({ Authorization: `Bearer ${token}` })

async function getProject(token, name) {
  // Возвращает активный проект по имени или null (не создаёт).
  const r = await api('/api/projects', { headers: auth(token) })
  if (r.status !== 200) fail(new Error(`GET /api/projects: ${r.status}`))
  const ps = Array.isArray(r.body) ? r.body : (r.body.projects || [])
  return ps.find((p) => p.name === name && !p.archived) || null
}

async function ensureProjects(token, names) {
  for (const name of names) {
    const p = await getProject(token, name)
    if (p) continue
    const meta = PROJECTS.find((x) => x.name === name) || {}
    const r = await api('/api/projects', {
      method: 'POST',
      headers: auth(token),
      body: JSON.stringify({ name, description: meta.description || 'e2e-fixture' })
    })
    if (r.status === 200 || r.status === 201) {
      console.log(`  project created: ${name}`)
    } else if (r.status === 409 || r.status === 400) {
      console.log(`  project skip (${r.status}): ${name}`)
    } else {
      fail(new Error(`POST /api/projects ${name}: ${r.status} ${JSON.stringify(r.body).slice(0, 200)}`))
    }
  }
}

async function ensureEpic(token, projectId, epicName) {
  // Возвращает {id, name} — существующий модуль или только что созданный.
  const r = await api(`/api/projects/${projectId}/epics`, { headers: auth(token) })
  if (r.status === 200 || r.status === 404) {
    const list = Array.isArray(r.body) ? r.body : (r.body.epics || [])
    const found = list.find((e) => e.name === epicName)
    if (found) return found
  }
  const c = await api(`/api/projects/${projectId}/epics`, {
    method: 'POST',
    headers: auth(token),
    body: JSON.stringify({ name: epicName })
  })
  if (c.status === 200 || c.status === 201) {
    console.log(`  epic created: ${epicName} (project ${projectId})`)
    return Array.isArray(c.body) ? c.body.find((e) => e.name === epicName) : c.body
  }
  fail(new Error(`POST epic «${epicName}»: ${c.status} ${JSON.stringify(c.body).slice(0, 200)}`))
}

function listTasks(r) {
  return Array.isArray(r.body) ? r.body : (r.body.tasks || [])
}

async function tasksOfProject(token, projectName) {
  const r = await api(`/api/tasks?project=${encodeURIComponent(projectName)}&per_page=200`, { headers: auth(token) })
  if (r.status !== 200) fail(new Error(`GET /api/tasks?project=${projectName}: ${r.status}`))
  return listTasks(r)
}

async function ensureTasks(token) {
  for (const fx of TASK_FIXTURES) {
    const p = await getProject(token, fx.project)
    if (!p) fail(new Error(`ensureTasks: проект «${fx.project}» не найден (сначала ensureProjects)`))

    let epicId = null
    if (fx.epic) {
      const epic = await ensureEpic(token, p.id, fx.epic)
      epicId = epic?.id
      if (!epicId) fail(new Error(`ensureTasks: epic_id для «${fx.epic}» не получен`))
    }

    const tasks = await tasksOfProject(token, fx.project)
    // Уже создана? (тот же epic_id — для epic-задач; null — для без-эпик.)
    const already = tasks.some((t) => (epicId ? t.epic_id === epicId : t.epic_id == null))
    if (already) {
      console.log(`  task exists: ${fx.project}${fx.epic ? '/' + fx.epic : ''}: ${fx.title}`)
      continue
    }

    const body = { title: fx.title, project: fx.project, description: 'e2e-фитур (seed-dev.mjs)' }
    if (epicId) body.epic_id = epicId
    const c = await api('/api/tasks', { method: 'POST', headers: auth(token), body: JSON.stringify(body) })
    if (c.status === 200 || c.status === 201) {
      console.log(`  task created: ${fx.project}${fx.epic ? '/' + fx.epic : ''}: ${fx.title}`)
    } else if (c.status === 409 || c.status === 400) {
      console.log(`  task skip (${c.status}): ${fx.title}`)
    } else {
      fail(new Error(`POST /api/tasks «${fx.title}»: ${c.status} ${JSON.stringify(c.body).slice(0, 200)}`))
    }
  }
}

async function main() {
  console.log(`seed-dev: target ${BASE}`)
  const token = await login()
  await ensureProjects(token, PROJECTS.map((p) => p.name))
  // Фиксированный модуль ТРК + задачи-фитуры (A/B/C прек-условия)
  await ensureTasks(token)
  console.log(`seed-dev: OK`)
}

main().then(() => process.exit(0)).catch(fail)
