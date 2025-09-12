import { beforeEach, describe, it, jest } from '@jest/globals'
import * as core from '../../test/mocks/core'
import { UpdateArgs } from '../../lib/trdl-cli'
import { trdlCli } from '../../test/mocks/trdl-cli'
import { getUpdateArgs, preset } from './preset'

// Mocks should be declared before the module being tested is imported.
jest.unstable_mockModule('@actions/core', () => core)

// The module being tested should be imported dynamically. This ensures that the
// mocks are used in place of any actual dependencies.
const { parseInputs, formatTrdlUseEnv, Do } = await import('./use')

describe('setup-app/src/use.ts', function () {
  describe('parseInputs', function () {
    it('should work if force is not passed and required=false and w/o optional fields', function () {
      core.getInput.mockReturnValue('')
      core.getBooleanInput.mockReturnValue(false)
      expect(parseInputs(false)).toEqual({
        force: false,
        repo: '',
        group: ''
      })
    })
    it('should work if force is passed and required=true', function () {
      const value = 'input value'
      core.getInput.mockReturnValue(value)
      core.getBooleanInput.mockReturnValue(true)
      expect(parseInputs(true)).toEqual({
        force: true,
        repo: value,
        group: value,
        channel: value
      })
    })
  })
  describe('formatTrdlUseEnv', function () {
    it('should work w/o channel', function () {
      const args: UpdateArgs = {
        repo: 'some repo',
        group: 'some group'
      }
      expect(formatTrdlUseEnv(args)).toEqual({
        key: 'TRDL_USE_SOME-REPO_GROUP_CHANNEL',
        value: `${args.group} `
      })
    })
    it('should work with channel', function () {
      const args: UpdateArgs = {
        repo: 'some repo',
        group: 'some group',
        channel: 'some channel'
      }
      expect(formatTrdlUseEnv(args)).toEqual({
        key: 'TRDL_USE_SOME-REPO_GROUP_CHANNEL',
        value: `${args.group} ${args.channel}`
      })
    })
  })
  describe('Do', function () {
    let updArgs: UpdateArgs
    beforeEach(function () {
      updArgs = { ...getUpdateArgs(preset.werf) }
    })
    it('should update app in background if preset=unknown and appPath is found', async function () {
      updArgs.repo = 'some repo'
      updArgs.group = 'some group'
      updArgs.channel = 'some channel'

      core.getInput.mockReturnValueOnce(updArgs.channel)
      core.getInput.mockReturnValueOnce(updArgs.repo)
      core.getInput.mockReturnValueOnce(updArgs.group)

      core.getBooleanInput.mockReturnValueOnce(false)

      const appPath = '/app/path'
      trdlCli.binPath.mockResolvedValueOnce(appPath)

      await Do(trdlCli, preset.unknown)

      expect(trdlCli.mustExist).toHaveBeenCalled()
      expect(trdlCli.binPath).toHaveBeenCalledWith(updArgs)
      expect(trdlCli.binPath).toHaveBeenCalledTimes(1)
      expect(trdlCli.update).toHaveBeenCalledWith(updArgs, { inBackground: true })
      expect(core.exportVariable).toHaveBeenCalledWith(
        'TRDL_USE_SOME-REPO_GROUP_CHANNEL',
        `${updArgs.group} ${updArgs.channel}`
      )
      expect(core.addPath).toHaveBeenCalledWith(appPath)
    })
    it('should update app in foreground if preset=werf and user inputs override group/channel', async function () {
      const userInputs = {
        group: 'user group',
        channel: 'user channel'
      }

      core.getInput.mockReturnValueOnce(userInputs.channel)
      core.getInput.mockReturnValueOnce('')
      core.getInput.mockReturnValueOnce(userInputs.group)
      core.getBooleanInput.mockReturnValueOnce(false)

      const appPath = '/app/path'
      trdlCli.binPath.mockResolvedValueOnce('')
      trdlCli.binPath.mockResolvedValueOnce(appPath)

      await Do(trdlCli, preset.werf)

      const presetArgs = getUpdateArgs(preset.werf)
      const expectedArgs = {
        ...presetArgs,
        group: userInputs.group,
        channel: userInputs.channel
      }

      expect(trdlCli.mustExist).toHaveBeenCalled()
      expect(trdlCli.binPath).toHaveBeenCalledWith(expectedArgs)
      expect(trdlCli.binPath).toHaveBeenCalledTimes(2)
      expect(trdlCli.update).toHaveBeenCalledWith(expectedArgs, { inBackground: false })
      expect(core.exportVariable).toHaveBeenCalledWith(
        'TRDL_USE_WERF_GROUP_CHANNEL',
        `${expectedArgs.group} ${expectedArgs.channel}`
      )
      expect(core.addPath).toHaveBeenCalledWith(appPath)
    })
  })
})
