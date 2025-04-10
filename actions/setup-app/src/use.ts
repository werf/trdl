import { addPath, debug, endGroup, exportVariable, getBooleanInput, getInput, info, startGroup } from '@actions/core'
import { TrdlCli, UpdateArgs } from '../../lib/trdl-cli'
import { getUpdateArgs, preset } from './preset'
import { optionalToObject } from '../../lib/optional'
import { format } from 'util'
import slugify from 'slugify'

interface inputs extends UpdateArgs {
  force: boolean
}

interface envVar {
  key: string
  value: string
}

function parseInputs(required: boolean): inputs {
  return {
    force: getBooleanInput('force', { required }),
    repo: getInput('repo', { required }),
    group: getInput('group', { required }),
    ...optionalToObject('channel', getInput('channel')) // optional field
  }
}

function mapInputsToCmdArgs(inputs: inputs): UpdateArgs {
  const { repo, group, channel } = inputs
  return {
    repo,
    group,
    ...optionalToObject('channel', channel) // optional field
  }
}

function formatTrdlUseEnv(args: UpdateArgs): envVar {
  const slugOpts = {
    strict: true
  }
  return {
    key: format('TRDL_USE_%s_GROUP_CHANNEL', slugify(args.repo, slugOpts)),
    value: format(`%s %s`, args.group, args.channel || '')
  }
}

export async function Do(trdlCli: TrdlCli, p: preset) {
  startGroup('Using application via "trdl update" and "trdl bin-path"')
  const noPreset = p === preset.unknown
  debug(format(`using preset=%s`, !noPreset))

  const inputs = parseInputs(noPreset)
  debug(format(`parsed inputs=%o`, inputs))

  const args = noPreset ? mapInputsToCmdArgs(inputs) : getUpdateArgs(p)
  debug(format(`merged(preset, inputs) args=%o`, args))

  await trdlCli.mustExist()

  let appPath = await trdlCli.binPath(args)
  debug(format(`"trdl bin-path" application path=%s`, appPath))

  if (!appPath) {
    const opts = { inBackground: false }
    info(format('Updating application via "trdl update" with args=%o and options=%o.', args, opts))
    await trdlCli.update(args, opts)

    appPath = await trdlCli.binPath(args)
    debug(format(`"trdl bin-path" application path=%s`, appPath))
  } else {
    const opts = { inBackground: true }
    info(format('Updating application via "trdl update" with args=%o and options=%o.', args, opts))

    await trdlCli.update(args, opts)
  }

  const trdlUseEnv = formatTrdlUseEnv(args)
  info(format('Exporting $%s=%s', trdlUseEnv.key, trdlUseEnv.value))
  exportVariable(trdlUseEnv.key, trdlUseEnv.value)

  info(format('Extending $PATH variable with app_path=%s', appPath))
  addPath(appPath)
  endGroup()
}
