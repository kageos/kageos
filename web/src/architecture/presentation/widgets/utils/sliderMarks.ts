const DEFAULT_MAX_LABELED_MARKS = 5

interface BuildSliderMarksOptions {
  min: number
  max: number
  step: number
  unit?: string
  maxLabeledMarks?: number
}

export function buildSliderMarks(options: BuildSliderMarksOptions): Record<number, string> {
  const min = normalizeFiniteNumber(options.min, 0)
  const max = normalizeFiniteNumber(options.max, 100)
  const step = normalizePositiveNumber(options.step, 1)
  const unit = options.unit || ''
  const maxLabeledMarks = Math.max(2, Math.floor(options.maxLabeledMarks || DEFAULT_MAX_LABELED_MARKS))

  if (max <= min) {
    return {
      [min]: formatMarkLabel(min, unit)
    }
  }

  const totalMarks = Math.round((max - min) / step) + 1
  const values = totalMarks <= maxLabeledMarks
    ? buildDenseMarkValues(min, max, step)
    : buildSparseMarkValues(min, max, step, maxLabeledMarks)

  const marks: Record<number, string> = {}
  values.forEach(value => {
    marks[value] = formatMarkLabel(value, unit)
  })
  return marks
}

function buildDenseMarkValues(min: number, max: number, step: number): number[] {
  const values: number[] = []
  for (let value = min; value < max; value += step) {
    values.push(roundToStepPrecision(value, step))
  }
  values.push(max)
  return ensureUniqueSortedValues(values)
}

function buildSparseMarkValues(min: number, max: number, step: number, maxLabeledMarks: number): number[] {
  const values = new Set<number>([min, max])
  const intervals = maxLabeledMarks - 1

  for (let index = 1; index < intervals; index += 1) {
    const ratio = index / intervals
    const rawValue = min + (max - min) * ratio
    const snapped = snapValueToStep(rawValue, min, max, step)
    values.add(snapped)
  }

  return ensureUniqueSortedValues(Array.from(values))
}

function snapValueToStep(value: number, min: number, max: number, step: number): number {
  const offset = Math.round((value - min) / step) * step
  const snapped = min + offset
  const clamped = Math.max(min, Math.min(max, snapped))
  return roundToStepPrecision(clamped, step)
}

function ensureUniqueSortedValues(values: number[]): number[] {
  return Array.from(new Set(values)).sort((left, right) => left - right)
}

function roundToStepPrecision(value: number, step: number): number {
  const stepString = String(step)
  const precision = stepString.includes('.') ? (stepString.split('.')[1]?.length ?? 0) : 0
  return Number(value.toFixed(precision))
}

function formatMarkLabel(value: number, unit: string): string {
  return unit ? `${value}${unit}` : String(value)
}

function normalizeFiniteNumber(value: number, fallback: number): number {
  return Number.isFinite(value) ? value : fallback
}

function normalizePositiveNumber(value: number, fallback: number): number {
  return Number.isFinite(value) && value > 0 ? value : fallback
}
