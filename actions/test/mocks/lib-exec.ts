import { jest } from '@jest/globals'
import type * as lib from '../../lib/exec'

export const execOutput = jest.fn<typeof lib.execOutput>()
