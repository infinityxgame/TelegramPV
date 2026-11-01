import { defineConfig, globalIgnores } from 'eslint/config'
import globals from 'globals'
import js from '@eslint/js'
import pluginVue from 'eslint-plugin-vue'
import prettierRecommended from 'eslint-plugin-prettier/recommended'

export default defineConfig([
  {
    name: 'app/files-to-lint',
    files: ['**/*.{vue,js,mjs,jsx}']
  },

  globalIgnores([
    '**/dist/**',
    '**/dist-ssr/**',
    '**/coverage/**',
    '**/wailsjs/**',
    '**/public/**'
  ]),

  {
    languageOptions: {
      globals: {
        ...globals.browser
      }
    }
  },

  {
    name: 'app/node-files',
    files: ['vite.config.js'],
    languageOptions: {
      globals: {
        ...globals.node
      }
    }
  },

  js.configs.recommended,
  ...pluginVue.configs['flat/recommended'],

  prettierRecommended
])
