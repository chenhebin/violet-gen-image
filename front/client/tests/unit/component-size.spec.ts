import {
  readFileSync,
  readdirSync,
  type Dirent,
} from 'node:fs'
import { resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const sourceDirectory = resolve(process.cwd(), 'src')

function collectVueFiles(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap(
    (entry: Dirent) => {
      const path = `${directory}/${entry.name}`
      if (entry.isDirectory()) return collectVueFiles(path)
      return entry.name.endsWith('.vue') ? [path] : []
    },
  )
}

describe('Vue component boundaries', () => {
  it('keeps every single-file component at or below 500 lines', () => {
    const oversized = collectVueFiles(sourceDirectory)
      .map((file) => ({
        file: file.replace(`${sourceDirectory}/`, ''),
        lines: readFileSync(file, 'utf8').split(/\r?\n/).length,
      }))
      .filter((item) => item.lines > 500)

    expect(oversized).toEqual([])
  })
})
