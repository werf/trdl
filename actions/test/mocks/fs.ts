import { jest } from '@jest/globals'
import type * as fs from 'node:fs'

export const chmodSync = jest.fn<typeof fs.chmodSync>()
