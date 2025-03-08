const path = require('path');
const TerserPlugin = require('terser-webpack-plugin');

module.exports = {
  mode: 'production',
  entry: './src/index.js',
  output: {
    filename: 'index.js',
    path: path.resolve(__dirname, 'dist')
  },
  optimization: {
    minimize: true, 
    minimizer: [
      new TerserPlugin({
        terserOptions: {
          compress: {
            drop_console: false, // Remove console statements
            drop_debugger: false, // Remove debugger statements
            passes: 3, // Number of times to pass the file for optimization
          },
          mangle: {
            toplevel: false, // Mangle top-level variables and functions
            module: false, // Mangle variables in modules
            keep_classnames: true, // Mangle class names
            keep_fnames: true, // Mangle function names
          },
          format: {
            beautify: true, // Disable beautification
          }
        },
        extractComments: false 
      }),
    ]
  },
  module: {
    rules: [
      /*{
        test: /\.js$/,
        exclude: /node_modules/,
        use: {
          loader: 'babel-loader',
          options: {
            presets: ['@babel/preset-env']
          },
        }
      },*/
      {
        test: /\.worker\.js$/,
        exclude: /node_modules/,
        use: [
          {
            loader: 'worker-loader',
            options: {
              filename: 'index.worker.js'
            }
          }
        ],
      },
    ]
  }
};
