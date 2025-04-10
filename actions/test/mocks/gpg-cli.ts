import { GpgCli } from '../../lib/gpg-cli'
import { jest } from '@jest/globals'

// instance
const cli = new GpgCli()

export const gpgCli = {
  name: cli.name,
  mustGnuGP: jest.fn<typeof cli.mustGnuGP>(),
  import: jest.fn<typeof cli.import>(),
  verify: jest.fn<typeof cli.verify>(),
  help: jest.fn<typeof cli.help>()
}
