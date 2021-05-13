// @ts-check
const path = require('path')

require('ts-node').register({
  transpileOnly: true,
  // Use config with "module": "commonjs" because not all modules involved in tasks are esnext modules.
  project: path.resolve(__dirname, './dev/tsconfig.json'),
})

const log = require('fancy-log')
const gulp = require('gulp')
const createWebpackCompiler = require('webpack')
const WebpackDevServer = require('webpack-dev-server')

const {
  graphQlSchema,
  graphQlOperations,
  schema,
  watchGraphQlSchema,
  watchGraphQlOperations,
  watchSchema,
} = require('../shared/gulpfile')

const webpackConfig = require('./webpack.config')

const WEBPACK_STATS_OPTIONS = {
  all: false,
  timings: true,
  errors: true,
  warnings: true,
  colors: true,
}

/**
 * @param {import('webpack').Stats} stats
 */
const logWebpackStats = stats => {
  log(stats.toString(WEBPACK_STATS_OPTIONS))
}

async function webpack() {
  const compiler = createWebpackCompiler(webpackConfig)
  /** @type {import('webpack').Stats} */
  const stats = await new Promise((resolve, reject) => {
    compiler.run((error, stats) => (error ? reject(error) : resolve(stats)))
  })
  logWebpackStats(stats)
  if (stats.hasErrors()) {
    throw Object.assign(new Error('Failed to compile'), { showStack: false })
  }
}

/**
 * Watch files and update the webpack bundle on disk without starting a dev server.
 */
async function watchWebpack() {
  const compiler = createWebpackCompiler(webpackConfig)
  compiler.hooks.watchRun.tap('Notify', () => log('Webpack compiling...'))
  await new Promise(() => {
    compiler.watch({ aggregateTimeout: 300 }, (error, stats) => {
      logWebpackStats(stats)
      if (error || stats.hasErrors()) {
        log.error('Webpack compilation error')
      } else {
        log('Webpack compilation done')
      }
    })
  })
}

async function webpackDevelopmentServer() {
  const sockHost = process.env.SOURCEGRAPH_HTTPS_DOMAIN || 'sourcegraph.test'
  const sockPort = Number(process.env.SOURCEGRAPH_HTTPS_PORT || 3443)

  /** @type {import('webpack-dev-server').Configuration & { liveReload?: boolean }} */
  const options = {
    hot: !process.env.NO_HOT,
    inline: !process.env.NO_HOT,
    allowedHosts: ['.host.docker.internal'],
    host: 'localhost',
    port: 3080,
    publicPath: '/.assets/',
    contentBase: './ui/assets',
    stats: WEBPACK_STATS_OPTIONS,
    noInfo: false,
    disableHostCheck: true,
    proxy: {
      '/': {
        target: 'http://localhost:3081',
        // Avoid crashing on "read ECONNRESET".
        onError: () => undefined,
        // Don't log proxy errors, these usually just contain
        // ECONNRESET errors caused by the browser cancelling
        // requests. This should not be needed to actually debug something.
        logLevel: 'silent',
        onProxyReqWs: (_proxyRequest, _request, socket) =>
          socket.on('error', error => console.error('WebSocket proxy error:', error)),
      },
    },
    sockHost,
    sockPort,
  }
  WebpackDevServer.addDevServerEntrypoints(webpackConfig, options)
  const server = new WebpackDevServer(createWebpackCompiler(webpackConfig), options)
  await new Promise((resolve, reject) => {
    server.listen(3080, '0.0.0.0', error => (error ? reject(error) : resolve()))
  })
}

// Ensure the typings that TypeScript depends on are build to avoid first-time-run errors
const codeGen = gulp.parallel(schema, graphQlOperations, graphQlSchema)

// Watches code generation only, rebuilds on file changes
const watchCodeGen = gulp.parallel(watchSchema, watchGraphQlSchema, watchGraphQlOperations)

/**
 * Builds everything.
 */
const build = gulp.series(codeGen, webpack)

/**
 * Starts a development server without initial code generation, watches everything and rebuilds on file changes.
 */
const developmentWithoutInitialCodeGen = gulp.parallel(watchCodeGen, webpackDevelopmentServer)

/**
 * Runs code generation first, then starts a development server, watches everything and rebuilds on file changes.
 */
const development = gulp.series(
  // Ensure the typings that TypeScript depends on are build to avoid first-time-run errors
  codeGen,
  developmentWithoutInitialCodeGen
)

/**
 * Watches everything, rebuilds on file changes and writes the bundle to disk.
 * Useful to running integration tests.
 */
const watch = gulp.series(
  // Ensure the typings that TypeScript depends on are build to avoid first-time-run errors
  codeGen,
  gulp.parallel(watchCodeGen, watchWebpack)
)

module.exports = {
  build,
  watch,
  dev: development,
  unsafeDev: developmentWithoutInitialCodeGen,
  webpackDevServer: webpackDevelopmentServer,
  webpack,
  watchWebpack,
}
