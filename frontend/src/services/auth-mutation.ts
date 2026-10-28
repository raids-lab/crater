// Authentication should report offline failures immediately instead of queuing
// credentials until connectivity returns. Each retry requires a new submission.
export const authMutationOptions = {
  networkMode: 'always',
  retry: false,
} as const
