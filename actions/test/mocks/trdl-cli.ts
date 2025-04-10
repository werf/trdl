import { TrdlCli } from '../../lib/trdl-cli'
import { jest } from '@jest/globals'

// instance
const cli = new TrdlCli()

export const trdlCli = {
  name: cli.name,
  defaults: jest.fn<typeof cli.defaults>(),
  mustExist: jest.fn<typeof cli.mustExist>(),
  add: jest.fn<typeof cli.add>(),
  remove: jest.fn<typeof cli.remove>(),
  update: jest.fn<typeof cli.update>(),
  binPath: jest.fn<typeof cli.binPath>(),
  list: jest.fn<typeof cli.list>()
}
