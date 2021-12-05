#!/usr/bin/env node
const { build } = require("esbuild");
const { solidPlugin } = require("esbuild-plugin-solid");

let watch = false;
let minify = true;

for (let i = 0; i < process.argv.length; i++) {
  switch (process.argv[i]) {
    case '--watch':
      watch = true;
    case '--no-minify':
      minify = false;
  }
}

build({
  entryPoints: ['./static/js/app.tsx'],
  bundle: true,
  outfile: 'static/out.js',
  minify: minify,
  watch: watch,
  logLevel: 'info',
  plugins: [solidPlugin()],
})
