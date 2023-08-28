const path = require('path');
const HtmlWebpackPlugin = require('html-webpack-plugin');

module.exports = {
    entry: './src/index.js',
    output: {
        filename: 'bundle.js',
        path: path.resolve(__dirname, 'dist')
    },
    resolve: {
        extensions: ['.js', '.jsx', '.ts', '.tsx'], // Prioritize .js and .jsx before .ts and .tsx
    },
    module: {
        rules: [
        {
            test: /\.(ts|tsx)$/,  // This rule will still transpile .ts and .tsx files using ts-loader
            use: 'ts-loader',
            exclude: /node_modules/,
        },
        {
            test: /\.(js|jsx)$/,  // This rule is for .js and .jsx files
            exclude: /node_modules/,
            use: {
            loader: "babel-loader",
            options: {
                presets: ["@babel/preset-env", "@babel/preset-react"]
            }
            }
        }
        ],
    },
    plugins: [
        new HtmlWebpackPlugin({
        template: './public/index.html'
        })
    ],
    devServer: {
        contentBase: path.join(__dirname, 'dist'),
        compress: true,
        port: 9000
    }
};