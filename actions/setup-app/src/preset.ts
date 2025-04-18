import { getInput } from '@actions/core'
import { AddArgs, UpdateArgs } from '../../lib/trdl-cli'

export enum preset {
  unknown = 'unknown',
  werf = 'werf',
  kubedog = 'kubedog',
  nelm = 'nelm'
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
  },
  [preset.nelm]: {
    repo: preset.nelm,
    url: 'https://storage.googleapis.com/nelm-tuf',
    rootVersion: '1',
    rootSha512:
      '2122fb476c48de4609fe6d3636759645996088ff6796857fc23ba4b8331a6e3a58fc40f1714c31bda64c709ef6f49bcc4691d091bad6cb1b9a631d8e06e1f308'
  }
}

const cmdUpdateArgsMap: Record<preset, UpdateArgs> = {
  [preset.unknown]: {
    repo: preset.unknown,
    group: ''
    // channel is optional field
  },
  [preset.werf]: {
    repo: preset.werf,
    group: '2',
    channel: 'stable'
  },
  [preset.kubedog]: {
    repo: preset.kubedog,
    group: '0',
    channel: 'stable'
  },
  [preset.nelm]: {
    repo: preset.nelm,
    group: '1',
    channel: 'stable'
  }
}

export function getAddArgs(presetVal: preset): AddArgs {
  return cmdAddArgsMap[presetVal]
}

export function getUpdateArgs(presetVal: preset): UpdateArgs {
  return cmdUpdateArgsMap[presetVal]
}

export function parsePresetInput(): preset {
  const p = (getInput('preset') as preset) || preset.unknown

  if (!(p in preset)) {
    throw new Error(`preset "${p}" not found. Available presets: ${Object.values(preset).join(', ')}`)
  }

  return p
}
