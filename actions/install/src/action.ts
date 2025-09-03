import { getInput, platform, addPath, info, startGroup, endGroup } from '@actions/core'
import { HttpClient } from '@actions/http-client'
import { downloadTool, find, cacheFile } from '@actions/tool-cache'
import { chmodSync } from 'node:fs'
import { join, dirname } from 'node:path'
import { GpgCli } from '../../lib/gpg-cli'
import { Defaults, TrdlCli } from '../../lib/trdl-cli'
import { format } from 'util'

interface inputs {
  channel?: string
  version?: string
}

interface options {
  channel: string
  version: string
}

export function parseInputs(): inputs {
  const channel = getInput('channel')
  const version = getInput('version')
  return {
    ...(channel !== '' ? { channel } : {}), // optional field
    ...(version !== '' ? { version } : {}) // optional field
  }
}

async function fetchVersion(group: string, channel: string): Promise<string> {
  const client = new HttpClient()
  const resp = await client.get(`https://tuf.trdl.dev/targets/channels/${group}/${channel}`)
  const version = await resp.readBody()
  return version.trim()
}

export async function getOptions(inputs: inputs, defaults: Defaults): Promise<options> {
  const channel = inputs.channel ?? defaults.channel
  const version = inputs.version ?? await fetchVersion(defaults.group, defaults.channel) // prettier-ignore

  return {
    channel,
    version
  }
}

export function formatDownloadUrls(version: string): string[] {
  // https://github.com/actions/toolkit/blob/main/packages/core/README.md#platform-helper
  const plat = translateNodeJSPlatformToTrdlPlatform(platform.platform)
  const arch = translateNodeJSArchToTrdlArch(platform.arch)
  const ext = platform.isWindows ? '.exe' : ''
  return [
    `https://tuf.trdl.dev/targets/releases/${version}/${plat}-${arch}/bin/trdl${ext}`, // bin
    `https://tuf.trdl.dev/targets/signatures/${version}/${plat}-${arch}/bin/trdl.sig`, // sig
    `https://trdl.dev/trdl-client.asc` // asc
  ]
}

function translateNodeJSPlatformToTrdlPlatform(platform: string): string {
  switch (platform) {
    case 'linux':
    case 'darwin':
      return platform
    case 'win32':
      return 'windows'
    default:
      throw new Error(`The platform ${platform} not supported`)
  }
}

function translateNodeJSArchToTrdlArch(arch: string): string {
  switch (arch) {
    case 'x64':
      return 'amd64'
    case 'arm64':
      return 'arm64'
    default:
      throw new Error(`The architecture ${arch} not supported`)
  }
}

async function downloadParallel(binUrl: string, sigUrl: string, ascUrl: string): Promise<string[]> {
  return Promise.all([
    // prettier-ignore
    downloadTool(binUrl),
    downloadTool(sigUrl),
    downloadTool(ascUrl)
  ])
}

function findTrdlCachePath(toolName: string, toolVersion: string): string {
  return find(toolName, toolVersion)
}

async function installTrdl(binPath: string, toolName: string, toolVersion: string): Promise<void> {
  // install tool
  const cachedPath = await cacheFile(binPath, toolName, toolName, toolVersion)
  const cachedFile = join(cachedPath, toolName)
  configureTrdl(cachedFile)
}

function configureTrdl(cachedFile: string): void {
  // set permissions
  chmodSync(cachedFile, 0o755)
  // add tool to $PATH
  addPath(dirname(cachedFile))
}

export async function Run(): Promise<void> {
  const trdlCli = new TrdlCli()
  const gpgCli = new GpgCli()
  const inputs = parseInputs()

  await Do(trdlCli, gpgCli, inputs)
}

export async function Do(trdlCli: TrdlCli, gpgCli: GpgCli, inputs: inputs): Promise<void> {
  startGroup(`Install or self-update ${trdlCli.name}.`)
  info(format(`Parsed inputs=%o`, inputs))

  const defaults = trdlCli.defaults()
  info(format(`${trdlCli.name} repository defaults=%o`, defaults))

  const options = await getOptions(inputs, defaults)
  info(format(`${trdlCli.name} installation options=%o`, options))

  const trdlCachePath = findTrdlCachePath(trdlCli.name, options.version)

  if (trdlCachePath) {
    info(`Downloading skipped. ${trdlCli.name}@v${options.version} is already found in dir = ${trdlCachePath}.`)

    info(`Configuring ${trdlCachePath} permissions and adding it to the $PATH.`)
    configureTrdl(join(trdlCachePath, trdlCli.name))

    info(`Verifying ${trdlCli.name} availability from $PATH.`)
    await trdlCli.mustExist()

    info(`Checking ${trdlCli.name} version before updating.`)
    await trdlCli.version()

    /*
    info(`Updating ${trdlCli.name} to group=${defaults.group} and channel=${defaults.channel}.`)
    await trdlCli.update(defaults)

    info(`Checking ${trdlCli.name} version after updating.`)
    await trdlCli.version()
    */

    endGroup()
    return
  }

  info(`Verifying ${gpgCli.name} availability from $PATH.`)
  await gpgCli.mustGnuGP()

  const [binUrl, sigUrl, ascUrl] = formatDownloadUrls(options.version)
  info(`${trdlCli.name} binUrl=${binUrl}`)
  info(`${trdlCli.name} sigUrl=${sigUrl}`)
  info(`${trdlCli.name} ascUrl=${ascUrl}`)

  info('Downloading binary and signatures.')
  const [binPath, sigPath, ascPath] = await downloadParallel(binUrl, sigUrl, ascUrl)

  info('Importing and verifying gpg keys.')
  await gpgCli.import(ascPath)
  await gpgCli.verify(sigPath, binPath)

  info(`Installing ${trdlCli.name} and adding it to the $PATH.`)
  await installTrdl(binPath, trdlCli.name, options.version)

  info(`Checking installed ${trdlCli.name} version.`)
  await trdlCli.version()
  endGroup()
}
