// Utilidad pura para comprobar si una dirección es de loopback.
export const isLoopbackHost = (host) => {
  if (typeof host !== 'string') return false
  let value = host.trim().toLowerCase()
  if (!value) return false

  const bracketed = value.match(/^\[([^\]]+)\]/)
  if (bracketed) {
    value = bracketed[1]
  } else {
    const ported = value.match(/^(.+):(\d+)$/)
    if (ported && !ported[1].includes(':')) value = ported[1]
  }

  const loopbackNames = ['localhost', '::1', '0:0:0:0:0:0:0:1']
  if (loopbackNames.includes(value)) return true

  const quads = value.split('.')
  if (quads.length === 4 && quads.every((q) => /^\d{1,3}$/.test(q))) {
    return Number(quads[0]) === 127
  }

  return false
}
