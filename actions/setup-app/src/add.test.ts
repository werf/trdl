import { beforeEach, describe, it, jest } from '@jest/globals'
import * as core from '../../test/mocks/core'
import { trdlCli } from '../../test/mocks/trdl-cli'
import { getAddArgs, preset } from './preset'
import { AddArgs, ListItem } from '../../lib/trdl-cli'

// Mocks should be declared before the module being tested is imported.
jest.unstable_mockModule('@actions/core', () => core)

// The module being tested should be imported dynamically. This ensures that the
// mocks are used in place of any actual dependencies.
const { parseInputs, Do } = await import('./add')

describe('setup-app/src/add.ts', function () {
  describe('parseInputs', function () {
    it('should work if force is not passed and required=false', function () {
      core.getInput.mockReturnValue('')
      core.getBooleanInput.mockReturnValue(false)
      expect(parseInputs(false)).toEqual({
        force: false,
        repo: '',
        url: '',
        rootVersion: '',
        rootSha512: ''
      })
    })
    it('should work if force is passed and required=true', function () {
      const value = 'input value'
      core.getInput.mockReturnValue(value)
      core.getBooleanInput.mockReturnValue(true)
      expect(parseInputs(true)).toEqual({
        force: true,
        repo: value,
        url: value,
        rootVersion: value,
        rootSha512: value
      })
    })
  })
  describe('Do', function () {
    let addArgs: AddArgs
    let listItem: ListItem
    beforeEach(function () {
      addArgs = getAddArgs(preset.werf)
      listItem = {
        name: addArgs.repo,
        url: addArgs.url,
        default: 'some default'
      }
    })
    it('should add app if preset=unknown and bin-path found', async function () {
      addArgs.repo = 'some repo'
      addArgs.url = 'some url'
      addArgs.rootVersion = 'some root version'
      addArgs.rootSha512 = 'some root sha512'

      core.getInput.mockReturnValueOnce(addArgs.repo)
      core.getInput.mockReturnValueOnce(addArgs.url)
      core.getInput.mockReturnValueOnce(addArgs.rootVersion)
      core.getInput.mockReturnValueOnce(addArgs.rootSha512)

      core.getBooleanInput.mockReturnValueOnce(false)

      trdlCli.list.mockResolvedValueOnce([])

      await Do(trdlCli, preset.unknown)

      expect(trdlCli.mustExist).toHaveBeenCalled()
      expect(trdlCli.list).toHaveBeenCalled()
      expect(trdlCli.add).toHaveBeenCalledWith(addArgs)
    })
    it('should throw err if preset=werf, app found, force=false and found.url != input.url', async function () {
      core.getInput.mockReturnValueOnce('')
      core.getBooleanInput.mockReturnValueOnce(false)

      listItem.url = 'another url'

      trdlCli.list.mockResolvedValueOnce([listItem])

      const expectedErr = new Error(
        `Already added repo.url=${listItem.url} is not matched with given input.url=${addArgs.url}. Use the force input to overwrite.`
      )
      await expect(Do(trdlCli, preset.werf)).rejects.toThrow(expectedErr)

      expect(trdlCli.mustExist).toHaveBeenCalled()
      expect(trdlCli.list).toHaveBeenCalled()
    })
    it('should do nothing if preset=werf, app found, force=false and found.url == input.url', async function () {
      core.getInput.mockReturnValue('')
      core.getBooleanInput.mockReturnValueOnce(false)

      trdlCli.list.mockResolvedValueOnce([listItem])

      await Do(trdlCli, preset.werf)

      expect(trdlCli.mustExist).toHaveBeenCalled()
      expect(trdlCli.list).toHaveBeenCalled()
      expect(trdlCli.add).not.toHaveBeenCalled()
    })
    it('should force add app if preset=werf, app found, force=true', async function () {
      core.getInput.mockReturnValue('')
      core.getBooleanInput.mockReturnValueOnce(true)

      trdlCli.list.mockResolvedValueOnce([listItem])

      await Do(trdlCli, preset.werf)

      expect(trdlCli.mustExist).toHaveBeenCalled()
      expect(trdlCli.list).toHaveBeenCalled()
      expect(trdlCli.remove).toHaveBeenCalledWith(addArgs)
      expect(trdlCli.add).toHaveBeenCalledWith(addArgs)
    })
  })
})
