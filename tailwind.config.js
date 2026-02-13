/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    "./templates/**/*.templ",
    "./templates/**/*_templ.go",
  ],
  theme: {
    extend: {
      fontFamily: { sans: ['"Inter"', 'system-ui', '-apple-system', 'sans-serif'] },
      boxShadow: { card: '0 1px 2px 0 rgb(0 0 0 / 0.03)' },
    },
  },
  plugins: [
    require('@tailwindcss/forms'),
  ],
}
