module.exports = {
  presets: [
    [
      '@babel/preset-env',
      {
        targets: {
          esmodules: true,
        },
      },
    ],
    [
      '@babel/preset-react',
      {
        runtime: 'automatic',
        importSource: 'preact',
      },
    ],
  ],
  plugins: [
    ['@babel/plugin-transform-react-jsx', {
      runtime: 'automatic',
      importSource: 'preact',
    }],
  ],
};
