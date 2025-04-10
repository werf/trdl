// See: https://rollupjs.org/introduction/

import path from 'node:path'
import commonjs from '@rollup/plugin-commonjs'
import nodeResolve from '@rollup/plugin-node-resolve'
import typescript from '@rollup/plugin-typescript'

export default [
  buildActionConfig('install')
]

function buildActionConfig (action) {
  return {
    input: path.join(action, 'src/index.ts'),
    output: {
      esModule: true,
      file: path.join(action, 'dist/index.js'),
      format: 'es',
      sourcemap: true
    },
    plugins: [
      typescript({
        compilerOptions: {
          // IMPORTANT NOTE
          // tsconfig.base.json configured to use >= Node14
          // https://github.com/tsconfig/bases?tab=readme-ov-file#table-of-tsconfigs
          // ---------------
          // Suppress warning: TS5110: Option 'module' must be set to 'Node16' when option 'moduleResolution' is set to 'Node16'
          // https://www.typescriptlang.org/tsconfig/#moduleResolution
          moduleResolution: 'bundler',
          // ---------------
          outDir: path.join(action, 'dist')
        }
      }),
      nodeResolve({ preferBuiltins: true }),
      commonjs()
    ]
  }
}
