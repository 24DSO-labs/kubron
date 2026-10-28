# Сравнение скорости обновления списков: до веб-версии и сейчас

Цель — получить цифры, а не ощущения: сколько миллисекунд занимает вызов бэкенда (например, `GetPods`) в старой и новой сборках
на одном и том же кластере и одних и тех же действиях.

## Что уже измерено (серверная часть, без сети)
`auth.Check` — проверка прав, которую делает каждый метод `App`: ~86 нс (коммит `7894962`, до веб-версии) против ~90 нс сейчас.
Разница в пределах шума и на 5–6 порядков меньше времени запроса к кластеру, поэтому слои авторизации/сессий на скорость списков не влияют.

## Как сравнить две сборки на вашей машине
1. Сделайте резервную копию `~/.kubron` (старая сборка может читать/писать файлы данных).
2. Старая сборка в отдельной папке:
   ```
   git worktree add ../kubron-old 7894962
   cd ../kubron-old/frontend && npm ci && cd .. && wails dev
   ```
   Текущая сборка — `wails dev` в основной папке. Запускайте по очереди, не одновременно.
3. В окне приложения откройте DevTools (в `wails dev` — правая кнопка → «Inspect»), вкладка Console, вставьте скрипт ниже.
4. Выполните одинаковый сценарий в обеих сборках: тот же кластер и namespace, раздел «Поды», 10 раз нажмите «Обновить»
   (или 10 раз переключитесь между namespace), затем в консоли `tReport()`.
5. Пришлите две таблицы (старая и новая).

```js
(() => {
  const app = window.go?.main?.App
  if (!app) return console.error('window.go.main.App не найден')
  const log = (window.__t = window.__t || [])
  for (const k of Object.keys(app)) {
    if (app[k].__wrapped) continue
    const orig = app[k]
    const wrapped = function (...a) {
      const t0 = performance.now()
      const r = orig.apply(this, a)
      Promise.resolve(r).catch(() => {}).finally(() => log.push([k, performance.now() - t0]))
      return r
    }
    wrapped.__wrapped = true
    app[k] = wrapped
  }
  const q = (xs, p) => xs[Math.min(xs.length - 1, Math.floor(xs.length * p))]
  window.tReset = () => { log.length = 0 }
  window.tReport = () => {
    const by = {}
    for (const [k, ms] of log) (by[k] ||= []).push(ms)
    console.table(Object.entries(by).map(([k, xs]) => {
      xs.sort((a, b) => a - b)
      return { method: k, n: xs.length, p50_ms: +q(xs, 0.5).toFixed(1), p95_ms: +q(xs, 0.95).toFixed(1), max_ms: +xs[xs.length - 1].toFixed(1) }
    }).sort((a, b) => b.p50_ms - a.p50_ms))
  }
  console.log('готово: tReset() — сбросить, tReport() — показать')
})()
```

## Как читать результат
- Время вызова выросло только у части методов (например, только у списков по нескольким namespace) — причина в этих методах, а не в общем слое.
- Время вызовов одинаково, а обновление «кажется» медленнее — причина во фронтенде (число вызовов, порядок/дебаунс обновлений, события
  `cluster:changed`), а не в бэкенде. Тогда сравните `n` — количество вызовов на одно действие.
- Время выросло у всех методов одинаково — ищем общую причину; тогда делим диапазон коммитов пополам (`git worktree add ../kubron-x <hash>`).
