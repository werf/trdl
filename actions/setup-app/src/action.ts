import { parsePresetInput } from './preset'
import { TrdlCli } from '../../lib/trdl-cli'
import { Do as DoInstall } from '../../install/src/action'
import { Do as DoAdd } from './add'
import { Do as DoUse } from './use'

export async function Run(): Promise<void> {
  const p = parsePresetInput()
  const cli = new TrdlCli()

  await DoInstall(cli, {})
  await DoAdd(cli, p)
  await DoUse(cli, p)
}
