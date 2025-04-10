import { beforeEach, describe, it, expect, jest } from '@jest/globals'
import * as io from '../test/mocks/io'
import * as libExec from '../test/mocks/lib-exec'
import type { ListItem, TrdlCli as TrdlCliType } from './trdl-cli'

// Mocks should be declared before the module being tested is imported.
jest.unstable_mockModule('@actions/io', () => io)
jest.unstable_mockModule('./exec', () => libExec)

// The module being tested should be imported dynamically. This ensures that the
// mocks are used in place of any actual dependencies.
const { TrdlCli } = await import('./trdl-cli')

describe('trdl-cli.ts', function () {
  let cli: TrdlCliType
  let cliName: string

  beforeEach(function () {
    cli = new TrdlCli()
    cliName = cli.defaults().repo
  })

  describe('defaults', function () {
    it('should work', function () {
      expect(cli.defaults()).toEqual({
        repo: cliName,
        group: '0',
        channel: 'stable'
      })
    })
  })

  describe('mustExist', function () {
    it('should not throw error if io.which does not throws an error', async function () {
      const result = await cli.mustExist()
      expect(result).toBeUndefined()
      expect(io.which).toHaveBeenCalledWith(cliName, true)
    })
    it('should throw error if io.which throws an error', async function () {
      const err0 = new Error('some err')
      io.which.mockRejectedValueOnce(err0)
      await expect(cli.mustExist()).rejects.toThrow(err0)
      expect(io.which).toHaveBeenCalledWith(cliName, true)
    })
  })

  describe('add', function () {
    it('should work', async function () {
      const repo = 'some repo'
      const url = 'some url'
      const rootSha512 = 'some root sha512'
      const rootVersion = 'some root version'
      const result = await cli.add({ repo, url, rootVersion, rootSha512 })
      expect(result).toBeUndefined()
      expect(libExec.execOutput).toHaveBeenCalledWith(cliName, ['add', repo, url, rootVersion, rootSha512])
    })
  })

  describe('remove', function () {
    it('should work', async function () {
      const repo = 'some repo'
      const result = await cli.remove({ repo })
      expect(result).toBeUndefined()
      expect(libExec.execOutput).toHaveBeenCalledWith(cliName, ['remove', repo])
    })
  })

  describe('update', function () {
    it('should work with required args and w/o options', async function () {
      const repo = 'some repo'
      const group = 'some group'
      const result = await cli.update({ repo, group })
      expect(result).toBeUndefined()
      expect(libExec.execOutput).toHaveBeenCalledWith(cliName, ['update', repo, group], { env: process.env })
    })
    it('should work with all args but w/o options', async function () {
      const repo = 'some repo'
      const group = 'some group'
      const channel = 'some channel'
      const result = await cli.update({ repo, group, channel })
      expect(result).toBeUndefined()
      expect(libExec.execOutput).toHaveBeenCalledWith(cliName, ['update', repo, group, channel], { env: process.env })
    })
    it('should work with all args and options', async function () {
      const repo = 'some repo'
      const group = 'some group'
      const channel = 'some channel'
      const inBackground = false
      const result = await cli.update({ repo, group, channel }, { inBackground })
      expect(result).toBeUndefined()
      const expectedEnv = { ...process.env, TRDL_IN_BACKGROUND: String(inBackground) }
      expect(libExec.execOutput).toHaveBeenCalledWith(cliName, ['update', repo, group, channel], { env: expectedEnv })
    })
  })

  describe('binPath', function () {
    it('should work with required args', async function () {
      const stdout = ['some', 'stdout']
      libExec.execOutput.mockResolvedValueOnce({ stdout, stderr: [], exitCode: 0 })
      const repo = 'some repo'
      const group = 'some group'
      const execOpts = {
        failOnStdErr: false,
        ignoreReturnCode: true
      }
      const result = await cli.binPath({ repo, group })
      expect(result).toEqual(stdout.join(''))
      expect(libExec.execOutput).toHaveBeenCalledWith(cliName, ['bin-path', repo, group], execOpts)
    })
    it('should work with all args', async function () {
      const stdout = ['some', 'stdout']
      libExec.execOutput.mockResolvedValueOnce({ stdout, stderr: [], exitCode: 0 })
      const repo = 'some repo'
      const group = 'some group'
      const channel = 'some channel'
      const execOpts = {
        failOnStdErr: false,
        ignoreReturnCode: true
      }
      const result = await cli.binPath({ repo, group, channel })
      expect(result).toEqual(stdout.join(''))
      expect(libExec.execOutput).toHaveBeenCalledWith(cliName, ['bin-path', repo, group, channel], execOpts)
    })
  })

  describe('list', function () {
    it('should work if underlined call return empty array', async function () {
      const stdout: string[] = []
      libExec.execOutput.mockResolvedValueOnce({ stdout, stderr: [], exitCode: 0 })
      const result = await cli.list()
      expect(result).toEqual(stdout)
      expect(libExec.execOutput).toHaveBeenCalledWith(cliName, ['list'])
    })
    it('should work if underlined call return array with item which has required fields', async function () {
      const stdout: string[] = ['Name  URL                   Default Channel ', 'trdl  https://tuf.trdl.dev  stable ']
      libExec.execOutput.mockResolvedValueOnce({ stdout, stderr: [], exitCode: 0 })
      const result = await cli.list()
      const item: ListItem = {
        name: 'trdl',
        url: 'https://tuf.trdl.dev',
        default: 'stable'
      }
      expect(result).toEqual([item])
      expect(libExec.execOutput).toHaveBeenCalledWith(cliName, ['list'])
    })
    it('should work if underlined call return array with item which all fields', async function () {
      const stdout: string[] = ['Name  URL                   Default Channel ', 'trdl  https://tuf.trdl.dev  stable 2 ']
      libExec.execOutput.mockResolvedValueOnce({ stdout, stderr: [], exitCode: 0 })
      const result = await cli.list()
      const item: ListItem = {
        name: 'trdl',
        url: 'https://tuf.trdl.dev',
        default: 'stable',
        channel: '2'
      }
      expect(result).toEqual([item])
      expect(libExec.execOutput).toHaveBeenCalledWith(cliName, ['list'])
    })
  })
  describe('version', function () {
    it('should work', async function () {
      const stdout: string[] = ['1.2.3']
      libExec.execOutput.mockResolvedValueOnce({ stdout, stderr: [], exitCode: 0 })
      const result = await cli.version()
      expect(result).toEqual(stdout.join(''))
      expect(libExec.execOutput).toHaveBeenCalledWith(cliName, ['version'], { silent: false })
    })
  })
})
