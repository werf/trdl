import { endGroup, getBooleanInput, getInput, info, startGroup } from '@actions/core'
import { AddArgs, TrdlCli } from '../../lib/trdl-cli'
import { getAddArgs, preset } from './preset'
import { format } from 'util'

interface inputs extends AddArgs {
  force: boolean
}

export function parseInputs(required: boolean): inputs {
  return {
    force: getBooleanInput('force'),
    repo: getInput('repo', { required }),
    url: getInput('url', { required }),
    rootVersion: getInput('root-version', { required }),
    rootSha512: getInput('root-sha512', { required })
  }
}

function mapInputsCmdArgs(inputs: inputs): AddArgs {
  const { repo, url, rootVersion, rootSha512 } = inputs
  return {
    repo,
    url,
    rootVersion,
    rootSha512
  }
}

export async function Do(trdlCli: TrdlCli, p: preset) {
  startGroup(`Adding application via "${trdlCli.name} add".`)

  const noPreset = p === preset.unknown
  info(format(`Using preset=%s.`, !noPreset))

  const inputs = parseInputs(noPreset)
  info(format(`Parsed inputs=%o.`, inputs))

  const args = noPreset ? mapInputsCmdArgs(inputs) : getAddArgs(p)
  info(format(`Options for finding and/or adding application=%o.`, args))

  info(`Verifying ${trdlCli.name} availability from $PATH.`)
  await trdlCli.mustExist()

  const list = await trdlCli.list()
  const found = list.find((item) => args.repo === item.name)

  if (!found) {
    info(`Application not found. Adding application via "${trdlCli.name} add".`)
    await trdlCli.add(args)
    endGroup()
    return
  }

  if (!inputs.force) {
    if (found.url !== args.url) {
      throw new Error(
        `Application is already added with repo.url=${found.url} which is not matched with given input.url=${args.url}. Use the force input to overwrite.`
      )
    }
    info(`Application addition skipped because it is already added with inputs.url=${args.url}.`)
    endGroup()
    return
  }

  // force adding
  info(`Force adding application using "${trdlCli.name} remove" and "${trdlCli.name} add".`)
  await trdlCli.remove(args)
  await trdlCli.add(args)
  endGroup()
}
