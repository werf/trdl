import { beforeEach, describe, it, expect, jest } from '@jest/globals'
import * as libExec from '../test/mocks/lib-exec'
import type { GpgCli as GpgCliType } from './gpg-cli'

// Mocks should be declared before the module being tested is imported.
jest.unstable_mockModule('./exec', () => libExec)

// The module being tested should be imported dynamically. This ensures that the
// mocks are used in place of any actual dependencies.
const { GpgCli } = await import('./gpg-cli')

describe('gpg-cli.ts', function () {
  let cli: GpgCliType
  let cliName: string

  beforeEach(function () {
    cli = new GpgCli()
    cliName = 'gpg'
  })

  describe('import', function () {
    it('should work', async function () {
      const ascPath = 'asc path'
      const result = await cli.import(ascPath)
      expect(result).toBeUndefined()
      expect(libExec.execOutput).toHaveBeenCalledWith(cliName, ['--import', ascPath])
    })
  })

  describe('verify', function () {
    it('should work', async function () {
      const sigPath = 'sig path'
      const binPath = 'bin path'
      const result = await cli.verify(sigPath, binPath)
      expect(result).toBeUndefined()
      expect(libExec.execOutput).toHaveBeenCalledWith(cliName, ['--verify', sigPath, binPath])
    })
  })

  describe('help', function () {
    it('should work', async function () {
      const stdout = ['some', 'stdout']
      libExec.execOutput.mockResolvedValueOnce({ stdout, stderr: [], exitCode: 0 })
      const result = await cli.help()
      expect(result).toEqual(stdout.join('\n'))
      expect(libExec.execOutput).toHaveBeenCalledWith(cliName, ['--help'])
    })
  })

  describe('mustGnuGP', function () {
    it('should throw the error if gpg is not GnuPG', async function () {
      const stdout = ['some', 'stdout']
      libExec.execOutput.mockResolvedValueOnce({ stdout, stderr: [], exitCode: 0 })
      await expect(cli.mustGnuGP()).rejects.toThrow(new Error('gpg is not GnuPG. Please install GnuPG'))
    })
    it('should work otherwise', async function () {
      const stdout = ['some', 'GnuPG', 'stdout']
      libExec.execOutput.mockResolvedValueOnce({ stdout, stderr: [], exitCode: 0 })
      const result = await cli.mustGnuGP()
      expect(result).toBeUndefined()
    })
  })
})
