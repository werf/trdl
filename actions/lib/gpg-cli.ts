import { execOutput } from './exec'

export class GpgCli {
  readonly name: string

  constructor() {
    this.name = 'gpg'
  }

  async mustGnuGP(): Promise<void> {
    const help = await this.help()
    if (!help.includes('GnuPG')) {
      throw new Error('gpg is not GnuPG. Please install GnuPG')
    }
  }

  async import(ascPath: string): Promise<void> {
    await execOutput(this.name, ['--import', ascPath], { failOnStdErr: false })
  }

  async verify(sigPath: string, binPath: string): Promise<void> {
    await execOutput(this.name, ['--verify', sigPath, binPath], { failOnStdErr: false })
  }

  async help(): Promise<string> {
    const { stdout } = await execOutput(this.name, ['--help'], { silent: true })
    return stdout.join('\n')
  }
}
