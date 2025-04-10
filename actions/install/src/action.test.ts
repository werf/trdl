import { describe, jest, it, beforeEach } from '@jest/globals'
import * as core from '../../test/mocks/core'
import * as toolCache from '../../test/mocks/tool-cache'
import * as fs from '../../test/mocks/fs'
import { trdlCli } from '../../test/mocks/trdl-cli'
import { gpgCli } from '../../test/mocks/gpg-cli'
import { join } from 'node:path'

// Mocks should be declared before the module being tested is imported.
jest.unstable_mockModule('@actions/core', () => core)
jest.unstable_mockModule('@actions/tool-cache', () => toolCache)
jest.unstable_mockModule('node:fs', () => fs)

// The module being tested should be imported dynamically. This ensures that the
// mocks are used in place of any actual dependencies.
const { parseInputs, getOptions, formatDownloadUrls, Do } = await import('./action')

describe('install/action.ts', function () {
  describe('parseInputs', function () {
    it('should work w/o required fields', function () {
      core.getInput.mockReturnValue('')
      expect(parseInputs()).toEqual({})
    })
    it('should work with all fields', function () {
      const channel = 'some channel'
      const version = 'some version'
      core.getInput.mockReturnValueOnce(channel)
      core.getInput.mockReturnValueOnce(version)
      expect(parseInputs()).toEqual({ channel, version })
    })
  })
  describe('getOptions', function () {
    const defaults = {
      repo: 'trdl',
      group: '0',
      channel: 'stable'
    }
    it('should work w/o required inputs', async function () {
      const opts = await getOptions({}, defaults)
      expect(opts).toHaveProperty('channel', defaults.channel)
      expect(opts).toHaveProperty('version')
      expect(opts.version).toMatch(/[0-9.]+/)
    })
    it('should work with all inputs', async function () {
      const inputs = {
        channel: 'some channel',
        version: 'some version'
      }
      const opts = await getOptions(inputs, defaults)
      expect(opts).toEqual(inputs)
    })
  })
  describe('formatDownloadUrls', function () {
    it('should work for platform=linux and arch=x64', function () {
      const version = '1.2.3'
      const plat = 'linux'
      const arch = 'amd64'
      const result = formatDownloadUrls(version)
      expect(result).toEqual([
        `https://tuf.trdl.dev/targets/releases/${version}/${plat}-${arch}/bin/trdl`,
        `https://tuf.trdl.dev/targets/signatures/${version}/${plat}-${arch}/bin/trdl.sig`,
        `https://trdl.dev/trdl-client.asc`
      ])
    })
  })
  describe('Do', function () {
    const inputs = {
      channel: 'test-channel',
      version: 'test-version'
    }
    const defaults = {
      repo: 'trdl',
      group: '0',
      channel: 'stable'
    }
    beforeEach(function () {
      trdlCli.defaults.mockReturnValue(defaults)
    })
    it('should not install trdl if tool cache is found', async function () {
      const someCache = '/path/to/tool'
      toolCache.find.mockReturnValueOnce(someCache)

      await Do(trdlCli, gpgCli, inputs)

      expect(toolCache.find).toHaveBeenCalledWith(trdlCli.name, inputs.version)
      expect(trdlCli.mustExist).toHaveBeenCalled()
      expect(trdlCli.version).toHaveBeenCalledTimes(2)
      expect(trdlCli.update).toHaveBeenCalledWith(defaults)
    })
    it('should install trdl if tool cache is not found', async function () {
      const binPath = '/tmp/cache/path'
      const sigPath = 'sig path'
      const ascPath = 'asc path'

      toolCache.downloadTool.mockResolvedValueOnce(binPath)
      toolCache.downloadTool.mockResolvedValueOnce(sigPath)
      toolCache.downloadTool.mockResolvedValueOnce(ascPath)

      const cachedPath = '/tmp/installed/path'
      toolCache.cacheFile.mockResolvedValueOnce(cachedPath)

      await Do(trdlCli, gpgCli, inputs)

      expect(toolCache.find).toHaveBeenCalledWith(trdlCli.name, inputs.version)
      expect(gpgCli.mustGnuGP).toHaveBeenCalled()
      expect(gpgCli.import).toHaveBeenCalledWith(ascPath)
      expect(gpgCli.verify).toHaveBeenCalledWith(sigPath, binPath)
      expect(toolCache.cacheFile).toHaveBeenCalledWith(binPath, trdlCli.name, trdlCli.name, inputs.version)
      expect(core.addPath).toHaveBeenCalledWith(cachedPath)
      expect(fs.chmodSync).toHaveBeenCalledWith(join(cachedPath, trdlCli.name), 0o755)
      expect(trdlCli.version).toHaveBeenCalled()
    })
  })
})
