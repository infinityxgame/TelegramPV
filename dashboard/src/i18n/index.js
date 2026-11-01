import { ref } from 'vue'
import en from './en.js'
import es from './es.js'

const messages = { en, es }
const currentLocale = ref('en')

function getNestedValue(obj, path) {
  return path
    .split('.')
    .reduce(
      (acc, part) => (acc && acc[part] !== undefined ? acc[part] : undefined),
      obj
    )
}

export function useI18n() {
  const setLocale = (lang) => {
    if (messages[lang]) {
      currentLocale.value = lang
    }
  }

  const t = (key, ...args) => {
    const localeDict = messages[currentLocale.value] || messages.en
    let str = getNestedValue(localeDict, key)
    if (str === undefined) {
      str = getNestedValue(messages.en, key) || key
    }

    if (typeof str === 'string' && args.length > 0) {
      args.forEach((arg, i) => {
        str = str.replace(new RegExp(`%${i + 1}|%s|%d`, 'g'), arg)
      })
    }
    return str
  }

  return {
    t,
    locale: currentLocale,
    setLocale
  }
}
