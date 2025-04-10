import { exec, ExecOptions } from '@actions/exec'

export interface ExecOutputResult {
  stdout: string[]
  stderr: string[]
  exitCode: number
}

export async function execOutput(
  commandLine: string,
  args?: string[],
  options?: ExecOptions
): Promise<ExecOutputResult> {
  const stdout: string[] = []
  const stderr: string[] = []

  const defaultOptions = {
    // https://github.com/actions/toolkit/blob/%40actions/exec%401.0.1/packages/exec/src/interfaces.ts#L39
    silent: true,
    failOnStdErr: true,
    listeners: {
      stdline(data: string) {
        stdout.push(data)
      },
      errline(data: string) {
        stderr.push(data)
      }
    }
  }

  const exitCode = await exec(commandLine, args, { ...defaultOptions, ...options })

  return {
    stdout,
    stderr,
    exitCode
  }
}
