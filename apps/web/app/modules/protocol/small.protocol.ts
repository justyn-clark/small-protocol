export const SMALL_PRIMITIVES = [
  "Schema",
  "Manifest",
  "Artifact",
  "Lineage",
  "Lifecycle",
] as const;

export const SMALL_RULES = {
  materializationRequiresValidation: true,
  artifactsAreImmutable: true,
  lineageIsAppendOnly: true,
  lifecycleIsEventBased: true,
  explicitContractsOnly: true,
} as const;

