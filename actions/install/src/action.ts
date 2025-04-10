import { Buffer } from 'node:buffer'
import { getInput, platform, addPath, info } from '@actions/core'
import { exec } from '@actions/exec'
import { HttpClient } from '@actions/http-client'
import { downloadTool, find, cacheFile } from '@actions/tool-cache'
import { chmodSync } from 'node:fs'

interface inputs {
  channel: string
  version: string
}

function parseInputs(): inputs {
  return {
    channel: getInput('channel'),
    version: getInput('version')
  }
}

async function fetchVersion(channel: string): Promise<string> {
  const client = new HttpClient()
  const resp = await client.get(`https://tuf.trdl.dev/targets/channels/0/${channel}`)
  const version = await resp.readBody()
  return version.trim()
}

async function getOptions(inputs: inputs): Promise<inputs> {
  const channel = inputs.channel || 'stable'
  const version = inputs.version || await fetchVersion(channel) // prettier-ignore

  return {
    channel,
    version
  }
}

function formatDownloadUrls(options: inputs): string[] {
  // https://github.com/actions/toolkit/blob/main/packages/core/README.md#platform-helper
  const plat = translateNodeJSPlatformToTrdlPlatform(platform.platform)
  const arch = translateNodeJSArchToTrdlArch(platform.arch)
  const ext = platform.isWindows ? '.exe' : ''
  const { version } = options
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

async function assertSystemGnuPG(): Promise<void> {
  // How to capture stdout
  // https://github.com/actions/toolkit/tree/main/packages/exec#outputoptions

  let stdout = Buffer.alloc(0)

  // https://github.com/actions/toolkit/blob/%40actions/exec%401.0.1/packages/exec/src/interfaces.ts
  const options = {
    silent: true,
    failOnStdErr: true,
    listeners: {
      stdout(data: Buffer) {
        stdout = Buffer.concat([stdout, data])
      }
    }
  }

  await exec('gpg', ['--help'], options)

  const isGnuGPG = stdout.toString().toLowerCase().includes('GnuPG'.toLowerCase())

  if (!isGnuGPG) {
    throw new Error('gpg is not GnuPG. Please install GnuPG')
  }
}

async function gpgVerify(binPath: string, sigPath: string, ascPath: string): Promise<void> {
  await assertSystemGnuPG()
  await exec('gpg', ['--import', ascPath])
  await exec('gpg', ['--verify', sigPath, binPath])
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
  const options = await getOptions(parseInputs())
  const [binUrl, sigUrl, ascUrl] = formatDownloadUrls(options)

  const toolName = 'trdl'
  const toolVersion = options.version

  const toolCache = findTrdlCache(toolName, toolVersion)
  if (toolCache) {
    info(`trdl@v${toolVersion} is found at path ${toolCache}. Installation skipped.`)
    return
  }

  const [binPath, sigPath, ascPath] = await downloadParallel(binUrl, sigUrl, ascUrl)
  await gpgVerify(binPath, sigPath, ascPath)

  await installTrdl(toolName, toolVersion, binPath)
}
