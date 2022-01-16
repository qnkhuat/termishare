module.exports = {
  important: true,
  purge: {
    // in prod look at shadow-cljs output file, in dev look at runtime, which will change files that are actually compiled; postcss watch should be a whole lot faster
    content: process.env.NODE_ENV == 'production' ? ["resources/frontend/static/main.js"] : ["resources/frontend/static/cljs-runtime/*.js"]
  },
  darkMode: false, // or 'media' or 'class'
  theme: {},
  variants: {},
  plugins: [],
}
