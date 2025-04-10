import { parsePresetInput } from './preset'
import { TrdlCli } from '../../lib/trdl-cli'
import { GpgCli } from '../../lib/gpg-cli'
import { Do as DoInstall } from '../../install/src/action'
import { Do as DoAdd } from './add'
import { Do as DoUse } from './use'

export async function Run(): Promise<void> {
  const p = parsePresetInput()
  const trdlCli = new TrdlCli()
  const gpgCli = new GpgCli()

  await DoInstall(trdlCli, gpgCli, {})
  await DoAdd(trdlCli, p)
  await DoUse(trdlCli, p)
}
