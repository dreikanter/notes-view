/** @type {import('tailwindcss').Config} */
module.exports = {
  content: [
    './web/static/index.html',
    './web/static/app.js',
  ],
  theme: {
    extend: {
      screens: {
        sidebar: '900px',
      },
    },
  },
  plugins: [require('@tailwindcss/typography')],
}
