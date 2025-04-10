/**
 * The entrypoint for the action. This file simply imports and runs the action.
 */
import { setFailed } from '@actions/core'
import { Run } from './action'

Run().catch(setFailed)
