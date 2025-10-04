// frontend/eslint.config.cjs
const js = require('@eslint/js')
const vue = require('eslint-plugin-vue')
const prettier = require('eslint-config-prettier')
const globals = require('globals')

// Vue flat preset may be an array or object
const vueFlat = vue.configs['flat/recommended'] || vue.configs.recommended
const vueConfigs = Array.isArray(vueFlat) ? vueFlat : [vueFlat]

module.exports = [
  // 1) Don’t lint build artifacts, deps, or this config file itself
  { ignores: ['dist/**', 'node_modules/**', 'eslint.config.cjs'] },

  // 2) Base JS rules
  js.configs.recommended,

  // 3) Vue rules
  ...vueConfigs,

  // 4) Disable rules that conflict with Prettier
  prettier,

  // 5) Project defaults for your source files
  {
    files: ['**/*.{js,vue}'],
    languageOptions: {
      ecmaVersion: 'latest',
      sourceType: 'module',
      // Browser globals (fetch, alert, window, document, etc.)
      globals: {
        ...globals.browser,
        // Explicit just in case your globals package is older:
        fetch: 'readonly',
        alert: 'readonly',
      },
    },
    rules: {
      'vue/multi-word-component-names': 'off',
    },
  },

  // 6) Optional: Node globals for config scripts if you want to lint them
  // Uncomment if you later want to lint Vite/Node config files:
  // {
  //   files: ['**/*.config.*', 'vite.config.*', '**/*.cjs'],
  //   languageOptions: { globals: globals.node },
  // }
]
