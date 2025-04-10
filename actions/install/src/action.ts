import { getInput, platform, addPath, info, startGroup, endGroup, debug } from '@actions/core'
import { HttpClient } from '@actions/http-client'
import { downloadTool, find, cacheFile } from '@actions/tool-cache'
import { chmodSync } from 'node:fs'
import { GpgCli } from '../../lib/gpg-cli'
import { Defaults, TrdlCli } from '../../lib/trdl-cli'
import { optionalToObject } from '../../lib/optional'
import { format } from 'util'

interface inputs {
  channel?: string
  version?: string
}

interface options {
  channel: string
  version: string
}

function parseInputs(): inputs {
  return {
    ...optionalToObject('channel', getInput('channel')), // optional field
    ...optionalToObject('version', getInput('version')) // optional field
  }
}

async function fetchVersion(group: string, channel: string): Promise<string> {
  const client = new HttpClient()
  const resp = await client.get(`https://tuf.trdl.dev/targets/channels/${group}/${channel}`)
  const version = await resp.readBody()
  return version.trim()
}

async function getOptions(inputs: inputs, defaults: Defaults): Promise<options> {
  const channel = inputs?.channel || defaults.channel
  const version = inputs?.version || await fetchVersion(defaults.group, defaults.channel) // prettier-ignore

  return {
    channel,
    version
  }
}

function formatDownloadUrls(version: string): string[] {
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

function findTrdlCache(toolName: string, toolVersion: string): string {
  return find(toolName, toolVersion)
}

async function installTrdl(toolName: string, toolVersion: string, binPath: string): Promise<void> {
  // install tool
  const installedPath = await cacheFile(binPath, toolName, toolName, toolVersion)
  // set permissions
  chmodSync(installedPath, 0o755)
  // add tool to $PATH
  addPath(installedPath)
}

export async function Run(): Promise<void> {
  const trdlCli = new TrdlCli()
  const inputs = parseInputs()

  await Do(trdlCli, inputs)
}

export async function Do(trdlCli: TrdlCli, inputs: inputs): Promise<void> {
  startGroup('Install or self-update trdl.')
  debug(format(`parsed inputs=%o`, inputs))

  const defaults = trdlCli.defaults()
  debug(format(`trdl defaults=%o`, defaults))

  const options = await getOptions(inputs, defaults)
  debug(format(`installation options=%o`, options))

  const toolCache = findTrdlCache(defaults.repo, options.version)

  if (toolCache) {
    info(`Installation skipped. trdl@v${options.version} is found at path ${toolCache}.`)

    await trdlCli.mustExist()
    info(`Updating trdl to group=${defaults.group} and channel=${defaults.channel}`)
    await trdlCli.update(defaults)

    endGroup()
    return
  }

  const gpgCli = new GpgCli()
  await gpgCli.mustGnuGP()

  const [binUrl, sigUrl, ascUrl] = formatDownloadUrls(options.version)
  debug(format('%s bin_url=%s', defaults.repo, binUrl))
  debug(format('%s sig_url=%s', defaults.repo, sigUrl))
  debug(format('%s asc_url=%s', defaults.repo, ascUrl))

  info('Downloading signatures.')
  const [binPath, sigPath, ascPath] = await downloadParallel(binUrl, sigUrl, ascUrl)

  info('Importing and verifying gpg keys.')
  await gpgCli.import(ascPath)
  await gpgCli.verify(sigPath, binPath)

  info('Installing trdl and adding it to the $PATH.')
  await installTrdl(defaults.repo, options.version, binPath)
  endGroup()
}
