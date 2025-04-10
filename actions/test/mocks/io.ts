import { jest } from '@jest/globals'
import type * as io from '@actions/io'

export const which = jest.fn<typeof io.which>()
