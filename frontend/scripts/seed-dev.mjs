// seed-dev.mjs — идемпотентный seed dev-БД для e2e (шаг 13.3)
// Поднимает бекенд dev (8081) и гарантирует наличие фиксированных проектов.
// Идемпотентность: проекты создаются, если их нет (API 409/400 — ignore).
// Запуск: node scripts/seed-dev.mjs [BASE_URL]
//   BASE_URL — http://127.0.0.1:5173 (через vite-прокси) или http://127.0.0.1:8081 (прямой бек)

const BASE = process.argv[2] || process.env.E2E_BASE_URL || 'http://127.0.0.1:5173'

const PROJECTS = [
  { name: 'Входящие', description: 'Fallback-проект' },
  { name: 'ТРК', description: 'Торговый комплекс' },
  { name: 'Отель', description: 'Отель/гостиница' },
  { name: 'Лидер Спорт', description: 'Фитнес' },
  { name: 'Театр', description: 'Театр' }
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

const auth = async (token) => ({ 'Authorization': `Bearer ${token}` })

async function ensureProjects(token) {
  // Сначала посмотрим, что уже есть
  const existing = await api('/api/projects', { headers: auth(token) })
  if (existing.status !== 200) {
    fail(new Error(`GET /api/projects: ${existing.status} ${JSON.stringify(existing.body).slice(0, 200)}`))
  }
  const active = (existing.body || []).filter((p) => !p.archived)
  const byName = new Map(active.map((p) => [p.name, p]))
  const missing = PROJECTS.filter((p) => !byName.has(p.name))
  if (missing.length === 0) {
    console.log(`seed-dev: все ${PROJECTS.length} проектов уже есть (идемпотентно, 0 новых)`)
    return
  }
  console.log(`seed-dev: создаю ${missing.length} проектов: ${missing.map((p) => p.name).join(', ')}`)
  for (const p of missing) {
    const r = await api('/api/projects', {
      method: 'POST',
      headers: auth(token),
      body: JSON.stringify(p)
    })
    if (r.status === 200 || r.status === 201) {
      console.log(`  created: ${p.name} (${r.status})`)
    } else if (r.status === 409 || r.status === 400) {
      // дубль или неверное имя — пропускаем
      console.log(`  skip (уже существует/конфликт ${r.status}): ${p.name}`)
    } else {
      console.warn(`  WARN ${p.name}: ${r.status} ${JSON.stringify(r.body).slice(0, 200)}`)
    }
  }
  // Финальная проверка
  const check = await api('/api/projects', { headers: auth(token) })
  const final = (check.body || []).filter((p) => !p.archived)
  const ok = PROJECTS.every((p) => final.some((x) => x.name === p.name))
  if (!ok) {
    fail(new Error('после seed: не все фиксированные проекты активны'))
  }
  console.log(`seed-dev: OK, активных проектов: ${final.length}`)
}

async function main() {
  console.log(`seed-dev: target ${BASE}`)
  const token = await login()
  await ensureProjects(token)
}

main().then(() => process.exit(0)).catch(fail)
