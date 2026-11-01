import { ref } from 'vue'
import en from './en.js'
import es from './es.js'

const messages = { en, es }
const currentLocale = ref('es')

function getNestedValue(obj, path) {
  return path
    .split('.')
    .reduce(
      (acc, part) => (acc && acc[part] !== undefined ? acc[part] : undefined),
      obj
    )
}

export function t(key, params, ...args) {
  const localeDict = messages[currentLocale.value] || messages.es
  let str = getNestedValue(localeDict, key)
  if (str === undefined) {
    str =
      getNestedValue(messages.es, key) ||
      getNestedValue(messages.en, key) ||
      key
  }

  if (typeof str !== 'string') return str

  if (params && typeof params === 'object' && !Array.isArray(params)) {
    str = str.replace(/\{(\w+)\}/g, (match, name) =>
      params[name] !== undefined && params[name] !== null
        ? String(params[name])
        : match
    )
  }

  if (args.length > 0) {
    args.forEach((arg, i) => {
      str = str.replace(new RegExp(`%${i + 1}|%s|%d`, 'g'), arg)
    })
  }
  return str
}

const SPLIT_MARK = '0000'
export function splitOn(key, mark, params = {}) {
  const parts = t(key, { ...params, [mark]: SPLIT_MARK }).split(SPLIT_MARK)
  return parts.length === 2 ? parts : [t(key, params), '']
}

export function useI18n() {
  const setLocale = (lang) => {
    if (messages[lang]) {
      currentLocale.value = lang
    }
  }

  return {
    t,
    splitOn,
    locale: currentLocale,
    setLocale
  }
}
