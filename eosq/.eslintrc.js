module.exports = {
  env: {
    browser: true,
    es6: true,
    jest: true
  },
  extends: [
    "react-app",
    "airbnb-typescript",
    "prettier",
    "prettier/@typescript-eslint",
    "prettier/react"
  ],
  globals: {
    Atomics: "readonly",
    SharedArrayBuffer: "readonly"
  },
  parserOptions: {
    ecmaFeatures: {
      jsx: true
    },
    ecmaVersion: 2018,
    sourceType: "module"
  },
  plugins: ["react"],
  rules: {
    "import/order": "off",
    "import/prefer-default-export": "off",
    "max-len": "off",
    "no-console": "off",
    "@typescript-eslint/no-unused-vars": "off",
    "react/destructuring-assignment": "off",
    "no-plusplus": "off",
    "default-case": "off",
    "lines-between-class-members": "off",
    "react/prop-types": "off",
    "react/sort-comp": "off",
    "react/jsx-boolean-value": "off",
    "react/static-property-placement": ["warn", "static public field"],
    "react/state-in-constructor": "off",
    "react/jsx-props-no-spreading": "off",
    "@typescript-eslint/no-use-before-define": "off",

    // List we should probably activate at some point
    "import/no-cycle": "off",
    "no-param-reassign": "off",
    "class-methods-use-this": "off",
    "react/no-array-index-key": "off",
    "react/jsx-no-target-blank": "off",
    "jsx-a11y/no-static-element-interactions": "off",
    "jsx-a11y/anchor-is-valid": "off",
    "jsx-a11y/click-events-have-key-events": "off",
    "jsx-a11y/alt-text": "off",
    "max-classes-per-file": "off",
    //disable warning for importing local symlinked packages
    "import/no-extraneous-dependencies": "off"
  }
}
