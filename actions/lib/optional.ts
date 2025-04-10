// optional array element
export function optionalToArray<Type>(arg: Type | undefined): Type[] {
  return arg ? [arg] : []
}

export function optionalToObject<Type>(key: string, value: Type | undefined): Record<string, Type> {
  return value ? { [key]: value } : {}
}
