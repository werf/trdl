import { jest } from '@jest/globals'
import type * as core from '@actions/core'

export const debug = jest.fn<typeof core.debug>()
// export const error = jest.fn<typeof core.error>()
export const info = jest.fn<typeof core.info>()
export const getInput = jest.fn<typeof core.getInput>()
export const getBooleanInput = jest.fn<typeof core.getBooleanInput>()
// export const setOutput = jest.fn<typeof core.setOutput>()
export const setFailed = jest.fn<typeof core.setFailed>()
export const addPath = jest.fn<typeof core.addPath>()
export const startGroup = jest.fn<typeof core.startGroup>()
export const endGroup = jest.fn<typeof core.endGroup>()
export const exportVariable = jest.fn<typeof core.exportVariable>()

export const platform = Object.create(
  {},
  {
    platform: {
      get: jest.fn().mockReturnValue('linux')
    },
    arch: {
      get: jest.fn().mockReturnValue('x64')
    },
    isWindows: {
      get: jest.fn().mockReturnValue(false)
    }
  }
)
