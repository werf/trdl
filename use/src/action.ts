import { ok } from 'node:assert/strict'
import { which } from '@actions/io'
import { getInput } from '@actions/core'
import { exec } from '@actions/exec'

interface inputs {
  repo: string
  group: string
  channel: string
  options: string
}

async function assertSystemTrdl(toolName: string): Promise<void> {
  await which(toolName, true)
}

function parseInputs(): inputs {
  return {
    repo: getInput('repo'),
    group: getInput('group'),
    channel: getInput('channel'),
    options: getInput('options')
  }
}

function assertInputs(inputs: inputs): void {
  ok(inputs.repo, 'repo param must be passed')
  ok(inputs.group, 'group param must be passed')
}

async function trdlUse(repo: string, group: string, channel: string, options: string): Promise<void> {
  await exec('trdl', ['use', repo, group, channel, options].filter(Boolean)) // ['x', ''].filter(Boolean) -> ['x']
}

export async function Run(): Promise<void> {
  const toolName = 'trdl'

  await assertSystemTrdl(toolName)

  const inputs = parseInputs()

  assertInputs(inputs)

  await trdlUse(inputs.repo, inputs.group, inputs.channel, inputs.options)
}
