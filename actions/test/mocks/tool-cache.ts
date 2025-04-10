import { jest } from '@jest/globals'
import type * as toolCache from '@actions/tool-cache'

export const downloadTool = jest.fn<typeof toolCache.downloadTool>()
export const find = jest.fn<typeof toolCache.find>()
export const cacheFile = jest.fn<typeof toolCache.cacheFile>()
