/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: 'class',
  content: [
    "./templates/**/*.templ",
    "./templates/**/*_templ.go",
  ],
  theme: {
    extend: {
      fontFamily: { sans: ['"Inter"', 'system-ui', '-apple-system', 'sans-serif'] },
      boxShadow: { card: '0 1px 3px 0 rgb(0 0 0 / 0.08), 0 4px 12px -2px rgb(0 0 0 / 0.08)' },
    },
  },
  plugins: [
    require('@tailwindcss/forms'),
  ],
}
