import { addPath, endGroup, exportVariable, getBooleanInput, getInput, info, startGroup } from '@actions/core'
import { TrdlCli, UpdateArgs } from '../../lib/trdl-cli'
import { getUpdateArgs, preset } from './preset'
import { format } from 'util'
import slugify from 'slugify'

interface inputs extends UpdateArgs {
  force: boolean
}

interface envVar {
  key: string
  value: string
}

export function parseInputs(required: boolean): inputs {
  const channel = getInput('channel')
  return {
    force: getBooleanInput('force'),
    repo: getInput('repo', { required }),
    group: getInput('group', { required }),
    ...(channel !== '' ? { channel } : {}) // optional field
  }
}

function mapInputsToCmdArgs(inputs: inputs): UpdateArgs {
  const { repo, group, channel } = inputs
  return {
    repo,
    group,
    ...(channel !== undefined ? { channel } : {}) // optional field
  }
}

export function formatTrdlUseEnv(args: UpdateArgs): envVar {
  const slugOpts = {
    strict: true
  }
  return {
    key: format('TRDL_USE_%s_GROUP_CHANNEL', slugify(args.repo, slugOpts).toUpperCase()),
    value: format(`%s %s`, args.group, args.channel || '')
  }
}

export async function Do(trdlCli: TrdlCli, p: preset) {
  startGroup(`Using application via "${trdlCli.name} update" and "${trdlCli.name} bin-path".`)
  const noPreset = p === preset.unknown
  info(format(`Using preset=%s`, !noPreset))

  const inputs = parseInputs(noPreset)
  info(format(`Parsed inputs=%o`, inputs))

  const args = noPreset ? mapInputsToCmdArgs(inputs) : getUpdateArgs(p)

  if (!noPreset) {
    if (inputs.group) args.group = inputs.group
    if (inputs.channel) args.channel = inputs.channel
  }

  info(format(`Options for using application=%o`, args))

  info(`Verifying ${trdlCli.name} availability from $PATH.`)
  await trdlCli.mustExist()

  let appPath = await trdlCli.binPath(args)
  info(`Found application path=${appPath}`)

  const hasAppPath = appPath !== ''

  const opts = { inBackground: hasAppPath }
  info(format(`Updating application via "${trdlCli.name} update" with options=%o`, opts))
  await trdlCli.update(args, opts)

  if (!hasAppPath) {
    appPath = await trdlCli.binPath(args)
    info(`Found application path=${appPath}`)
  }

  const trdlUseEnv = formatTrdlUseEnv(args)
  info(format('Exporting variable $%s=%s', trdlUseEnv.key, trdlUseEnv.value))
  exportVariable(trdlUseEnv.key, trdlUseEnv.value)

  info(`Extending $PATH variable with app_path=${appPath}`)
  addPath(appPath)
  endGroup()
}
