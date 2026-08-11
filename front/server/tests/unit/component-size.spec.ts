import { readdirSync, readFileSync } from 'node:fs'
import { extname, join, relative, resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const sourceRoot = resolve(process.cwd(), 'src')

function collectVueFiles(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name)
    return entry.isDirectory()
      ? collectVueFiles(path)
      : extname(entry.name) === '.vue'
        ? [path]
        : []
  })
}

describe('Vue component size', () => {
  it('keeps every single-file component at or below 500 lines', () => {
    const oversized = collectVueFiles(sourceRoot)
      .map((path) => ({
        path: relative(sourceRoot, path),
        lines: readFileSync(path, 'utf8').split(/\r?\n/u).length,
      }))
      .filter((file) => file.lines > 500)

    expect(oversized, JSON.stringify(oversized, null, 2)).toEqual([])
  })
})
