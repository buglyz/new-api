/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { spawnSync } from 'node:child_process'
import { existsSync, readFileSync } from 'node:fs'
import { extname, resolve } from 'node:path'

const mode = process.argv[2]
const modes = new Set(['--lint', '--format', '--copyright'])
if (!modes.has(mode)) {
  console.error(
    'Usage: node scripts/check-changed-quality.mjs --lint|--format|--copyright'
  )
  process.exit(2)
}

const webRoot = process.cwd()
const repoRoot = resolve(webRoot, '..')
const headSha = process.env.HEAD_SHA || 'HEAD'
const emptyTreeSha = '4b825dc642cb6eb9a060e54bf8d69288fbee4904'

function runGit(args) {
  const result = spawnSync('git', args, { cwd: repoRoot, encoding: 'utf8' })
  if (result.status !== 0) return null
  return result.stdout
}

function gitSucceeds(args) {
  return spawnSync('git', args, { cwd: repoRoot }).status === 0
}

function resolveBaseSha() {
  const candidate = process.env.BASE_SHA
  if (
    candidate &&
    !/^0+$/.test(candidate) &&
    gitSucceeds(['cat-file', '-e', `${candidate}^{commit}`])
  ) {
    return candidate
  }
  return (
    runGit(['rev-parse', '--verify', `${headSha}^`])?.trim() || emptyTreeSha
  )
}

const baseSha = resolveBaseSha()
const changedOutput = runGit([
  'diff',
  '--name-only',
  '-z',
  baseSha,
  headSha,
  '--',
  'web',
])
if (changedOutput == null) {
  console.error(`Unable to compare ${baseSha} with ${headSha}`)
  process.exit(1)
}

const changedFiles = [
  ...new Set(
    changedOutput
      .split('\0')
      .filter((file) => file.startsWith('web/'))
      .map((file) => file.slice('web/'.length))
      .filter((file) => file && existsSync(resolve(webRoot, file)))
  ),
]

const lintExtensions = new Set(['.cjs', '.js', '.jsx', '.mjs', '.ts', '.tsx'])
const formatExtensions = new Set([...lintExtensions, '.css', '.json', '.scss'])
const sourceExtensions = new Set([...lintExtensions, '.css', '.scss'])

function filesFor(extensions) {
  return changedFiles.filter((file) => extensions.has(extname(file)))
}

function runTool(tool, args) {
  const suffix = process.platform === 'win32' ? '.cmd' : ''
  const executable = resolve(
    webRoot,
    'node_modules',
    '.bin',
    `${tool}${suffix}`
  )
  const result = spawnSync(executable, args, { cwd: webRoot, stdio: 'inherit' })
  if (result.error) {
    console.error(`Unable to run ${tool}: ${result.error.message}`)
    return 1
  }
  return result.status ?? 1
}

let files
switch (mode) {
  case '--lint':
    files = filesFor(lintExtensions)
    if (files.length > 0) {
      process.exit(runTool('oxlint', ['-c', '.oxlintrc.json', ...files]))
    }
    break
  case '--format':
    files = filesFor(formatExtensions)
    if (files.length > 0) {
      process.exit(
        runTool('oxfmt', [
          '-c',
          '.oxfmtrc.json',
          '--ignore-path',
          '.gitignore',
          '--check',
          ...files,
        ])
      )
    }
    break
  case '--copyright': {
    files = filesFor(sourceExtensions).filter(
      (file) => file.startsWith('src/') || file.startsWith('scripts/')
    )
    const projectHeader =
      /^\/\*\r?\nCopyright \(C\) [\s\S]*?QuantumNous[\s\S]*?\*\/\r?\n?/
    const thirdPartyHeader = /^\/\*[\s\S]*?Copyright[\s\S]*?\*\/\r?\n?/i
    const generatedMarkers = [
      'This file was automatically generated',
      'This file is auto-generated',
      'This file is generated',
      'DO NOT EDIT',
      'You should NOT make any changes in this file',
    ]
    const missing = files.filter((file) => {
      const text = readFileSync(resolve(webRoot, file), 'utf8')
      if (generatedMarkers.some((marker) => text.includes(marker))) return false
      if (projectHeader.test(text)) return false
      return !thirdPartyHeader.test(text)
    })
    if (missing.length > 0) {
      console.error('Copyright headers missing from changed files:')
      missing.forEach((file) => console.error(`- ${file}`))
      process.exit(1)
    }
    break
  }
}

console.log(
  `Changed frontend ${mode.slice(2)} check passed (${files.length} file(s)).`
)
