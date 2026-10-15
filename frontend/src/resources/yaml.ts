/** Построители YAML-строк для блока "Affinity" в панелях (рисует CodeBlockSection) */

const sp = (n: number) => '  '.repeat(n)

export function buildPodAntiAffinityYaml(a: any): string[] {
  if (!a) return []
  const out: string[] = []

  const pref = a.preferredDuringSchedulingIgnoredDuringExecution
  if (pref?.length) {
    out.push('preferredDuringSchedulingIgnoredDuringExecution:')
    pref.forEach((t: any) => {
      out.push(sp(1) + '- weight: ' + t.weight)
      out.push(sp(2) + 'podAffinityTerm:')
      out.push(sp(3) + 'labelSelector:')
      out.push(sp(4) + 'matchLabels:')
      const ml = t.podAffinityTerm?.labelSelector?.matchLabels || {}
      for (const [k, v] of Object.entries(ml)) out.push(sp(5) + k + ': ' + v)
      out.push(sp(3) + 'topologyKey: ' + (t.podAffinityTerm?.topologyKey || ''))
    })
  }

  const req = a.requiredDuringSchedulingIgnoredDuringExecution
  if (req?.length) {
    out.push('requiredDuringSchedulingIgnoredDuringExecution:')
    req.forEach((t: any) => {
      out.push(sp(1) + '- labelSelector:')
      out.push(sp(2) + 'matchLabels:')
      const ml = t.podAffinityTerm?.labelSelector?.matchLabels || {}
      for (const [k, v] of Object.entries(ml)) out.push(sp(3) + k + ': ' + v)
      out.push(sp(2) + 'topologyKey: ' + (t.podAffinityTerm?.topologyKey || ''))
    })
  }
  return out
}

export function buildNodeAffinityYaml(affinity: any): string[] {
  const na = affinity?.nodeAffinity
  if (!na) return []
  const out: string[] = []

  const req = na.requiredDuringSchedulingIgnoredDuringExecution?.nodeSelectorTerms
  if (req) {
    out.push('requiredDuringSchedulingIgnoredDuringExecution:')
    out.push(sp(1) + 'nodeSelectorTerms:')
    req.forEach((term: any) => {
      if (term.matchExpressions?.length) {
        out.push(sp(2) + '- matchExpressions:')
        term.matchExpressions.forEach((expr: any) => {
          out.push(sp(3) + '- key: ' + expr.key)
          out.push(sp(4) + 'operator: ' + expr.operator)
          if (expr.values?.length) {
            out.push(sp(4) + 'values:')
            expr.values.forEach((val: string) => out.push(sp(5) + '- ' + val))
          }
        })
      }
    })
  }

  const pref = na.preferredDuringSchedulingIgnoredDuringExecution
  if (pref?.length) {
    if (out.length) out.push('')
    out.push('preferredDuringSchedulingIgnoredDuringExecution:')
    pref.forEach((item: any) => {
      out.push(sp(1) + '- weight: ' + item.weight)
      out.push(sp(2) + 'preference:')
      if (item.preference?.matchExpressions?.length) {
        out.push(sp(3) + 'matchExpressions:')
        item.preference.matchExpressions.forEach((expr: any) => {
          out.push(sp(4) + '- key: ' + expr.key)
          out.push(sp(5) + 'operator: ' + expr.operator)
          if (expr.values?.length) {
            out.push(sp(5) + 'values:')
            expr.values.forEach((val: string) => out.push(sp(6) + '- ' + val))
          }
        })
      }
    })
  }
  return out
}
