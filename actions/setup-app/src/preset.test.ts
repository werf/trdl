import { describe, jest, it } from '@jest/globals'
import * as core from '../../test/mocks/core'

// Mocks should be declared before the module being tested is imported.
jest.unstable_mockModule('@actions/core', () => core)

// The module being tested should be imported dynamically. This ensures that the
// mocks are used in place of any actual dependencies.
const { preset, parsePresetInput } = await import('./preset')

describe('setup-app/src/preset.ts', function () {
  describe('parsePresetInput', function () {
    it('should return unknown if preset not passed', function () {
      core.getInput.mockReturnValueOnce('')

      const p = parsePresetInput()
      expect(p).toEqual(preset.unknown)
    })
    it('should throw err if preset is passed but not in enum', function () {
      const p = 'some'
      core.getInput.mockReturnValueOnce(p)
      expect(parsePresetInput).toThrow(
        `preset "${p}" not found. Available presets: ${Object.values(preset).join(', ')}`
      )
    })
    it('should return preset otherwise', function () {
      const p = 'werf'
      core.getInput.mockReturnValueOnce(p)
      expect(parsePresetInput()).toEqual(p)
    })
  })
})
