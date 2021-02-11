const CracoLessPlugin = require("craco-less")
const { join } = require("path")

const nodeModules = (element) => join(__dirname, "node_modules", element)

module.exports = {
  webpack: {
    alias: {
      // Let's ensure we resovle to our own copy of React since when building for production,
      // it seems we have multiple copies and if linking `yarn link @dfuse/explorer` also. So
      // whatever the environment, let's resolve all `react` import to our own.
      react: nodeModules("react"),
    },
  },

  plugins: [
    {
      plugin: CracoLessPlugin,
      options: {
        lessLoaderOptions: {
          lessOptions: {
            modifyVars: {
              "@progress-default-color": "#7d90ff",
              "@progress-remaining-color": "#273a4c",
              "@progress-radius": "0px",
              "@tooltip-bg": "hsl(209, 32%, 31%)",
            },
            javascriptEnabled: true,
          },
        },
      },
    },
  ],
}