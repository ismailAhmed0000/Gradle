import { ApiError } from './client'

type FetchResult<T> = {
  data?: T
  error?: { error?: string }
  response: Response
}

export function unwrap<T>({ data, error, response }: FetchResult<T>): T {
  if (error !== undefined || !response.ok) {
    throw new ApiError(error?.error ?? `request failed with status ${response.status}`)
  }
  if (data === undefined) {
    throw new ApiError('request succeeded but returned no data')
  }
  return data
}
