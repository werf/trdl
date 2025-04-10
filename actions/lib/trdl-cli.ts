import { which } from '@actions/io'
import { execOutput } from './exec'

export class TrdlCli {
  readonly name: string

  constructor() {
    this.name = 'trdl'
  }

  defaults(): Defaults {
    return {
      repo: this.name,
      group: '0',
      channel: 'stable'
    }
  }

  // throws the error if trdl is not exist
  async mustExist(): Promise<void> {
    await which(this.name, true)
  }

  async add(args: AddArgs): Promise<void> {
    const { repo, url, rootVersion, rootSha512 } = args
    await execOutput(this.name, ['add', repo, url, rootVersion, rootSha512])
  }

  async remove(args: RemoveArgs): Promise<void> {
    const { repo } = args
    await execOutput(this.name, ['remove', repo])
  }

  async update(args: UpdateArgs, opts?: UpdateOptions) {
    const { repo, group, channel } = args
    const env = { ...(process.env as execOptionsEnvs), ...(opts && toUpdateEnvs(opts)) }
    const channelOpt = channel !== undefined ? [channel] : [] // optional field
    await execOutput(this.name, ['update', repo, group, ...channelOpt], { env })
  }

  async binPath(args: UpdateArgs): Promise<string> {
    const { repo, group, channel } = args
    const execOpts = {
      failOnStdErr: false,
      ignoreReturnCode: true
    }
    const channelOpt = channel !== undefined ? [channel] : [] // optional field
    const { stdout } = await execOutput(this.name, ['bin-path', repo, group, ...channelOpt], execOpts)
    return stdout.join('')
  }

  async list(): Promise<ListItem[]> {
    const { stdout } = await execOutput(this.name, ['list'])
    return stdout.slice(1).map(parseLineToItem)
  }
}

export interface AddArgs {
  repo: string
  url: string
  rootVersion: string
  rootSha512: string
}

export interface RemoveArgs {
  repo: string
}

export interface UpdateArgs {
  repo: string
  group: string
  channel?: string
}

export interface UpdateOptions {
  inBackground: boolean
}

export interface Defaults extends UpdateArgs {
  channel: string
}

export interface ListItem {
  name: string
  url: string
  default: string
  channel?: string
}

function parseLineToItem(line: string): ListItem {
  const [name, url, default_, channel] = line.trim().split(/ +/)
  return {
    name,
    url,
    default: default_,
    ...(channel !== undefined ? { channel } : {}) // optional field
  }
}

interface execOptionsEnvs {
  [key: string]: string
}

function toUpdateEnvs(opts: UpdateOptions): execOptionsEnvs {
  const env: execOptionsEnvs = {}
  // eslint-disable-next-line no-prototype-builtins
  if (opts.hasOwnProperty('inBackground')) {
    env['TRDL_IN_BACKGROUND'] = String(opts.inBackground)
  }
  return env
}
