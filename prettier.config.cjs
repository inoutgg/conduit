/** @type {import('prettier').Options} */
module.exports = {
  plugins: ["prettier-plugin-sql-cst"],
  overrides: [
    {
      files: ["*.sql"],
      options: {
        parser: "postgresql",
        sqlCanonicalSyntax: true,
        sqlKeywordCase: "upper",
        sqlParamTypes: ["?", "$nr", "@name"],
      },
    },
  ],
  trailingComma: "none",
  singleQuote: false,
  semi: true,
  tabWidth: 2,
  bracketSpacing: true,
  arrowParens: "always",
  quoteProps: "preserve",
  endOfLine: "lf",
};
