import { AxiosError } from 'axios';
import type { ErrorResponse } from '../types/api';

export function getApiErrorMessage(error: unknown, fallback: string) {
  const axiosError = error as AxiosError<ErrorResponse>;
  return axiosError.response?.data?.error?.message || fallback;
}
