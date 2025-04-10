import { getInput } from '@actions/core'
import { AddArgs, BinPathArgs, UpdateArgs } from '../../lib/trdl-cli'

export enum preset {
  unknown = 'unknown',
  werf = 'werf',
  kubedog = 'kubedog'
}

const cmdAddArgsMap: Record<preset, AddArgs> = {
  [preset.unknown]: {
    repo: preset.unknown,
    url: '',
    rootVersion: '',
    rootSha512: ''
  },
  [preset.werf]: {
    repo: preset.werf,
    url: 'https://tuf.werf.io',
    rootVersion: '12',
    rootSha512:
      'e1d3c7bcfdf473fe1466c5e9d9030bea0fed857d0563db1407754d2795256e4d063b099156807346cdcdc21d747326cc43f96fa2cacda5f1c67c8349fe09894d'
  },
  [preset.kubedog]: {
    repo: preset.kubedog,
    url: 'https://tuf.kubedog.werf.io',
    rootVersion: '12',
    rootSha512:
      '6462a80292eb6d7712d8a18126366511f9c47a566f121a7745cfd68b624dc340b6591c2cadfe20690eb38296c399a3f4e6948aca90be60e446ed05c3c238294c'
  }
}

const cmdUpdateArgsMap: Record<preset, UpdateArgs | BinPathArgs> = {
  [preset.unknown]: {
    repo: preset.unknown,
    group: ''
    // channel is optional field
  },
  [preset.werf]: {
    repo: preset.werf,
    group: 'stable',
    channel: '2'
  },
  [preset.kubedog]: {
    repo: preset.kubedog,
    group: 'stable',
    channel: '0'
  }
}

export function getAddArgs(presetVal: preset): AddArgs {
  return cmdAddArgsMap[presetVal]
}

export function getUpdateArgs(presetVal: preset): UpdateArgs | BinPathArgs {
  return cmdUpdateArgsMap[presetVal]
}

export function parsePresetInput(): preset {
  const p = (getInput('preset') as preset) || preset.unknown

  if (!(p in preset)) {
    throw new Error(`preset "${p}" not found. Available presets: ${Object.values(preset).join(' ,')}`)
  }

  return p
}
