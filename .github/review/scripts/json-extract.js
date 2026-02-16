'use strict'

function isPlainObject(value) {
  return Boolean(value && typeof value === 'object' && !Array.isArray(value))
}

function parseObjectCandidate(candidate) {
  try {
    const parsed = JSON.parse(candidate)
    return isPlainObject(parsed) ? parsed : null
  } catch {
    return null
  }
}

function looksLikeReviewPayload(value) {
  return isPlainObject(value) && ('verdict' in value || 'summary' in value || 'findings' in value)
}

function collectJsonObjectCandidates(text) {
  const candidates = []
  let depth = 0
  let inString = false
  let escaped = false
  let currentStart = -1

  for (let index = 0; index < text.length; index += 1) {
    const char = text[index]

    if (inString) {
      if (escaped) {
        escaped = false
      } else if (char === '\\') {
        escaped = true
      } else if (char === '"') {
        inString = false
      }
      continue
    }

    if (char === '"') {
      inString = true
      continue
    }

    if (char === '{') {
      if (depth === 0) currentStart = index
      depth += 1
      continue
    }

    if (char === '}') {
      if (depth === 0) {
        currentStart = -1
        continue
      }

      depth -= 1
      if (depth === 0 && currentStart !== -1) {
        const candidate = text.slice(currentStart, index + 1).trim()
        const parsed = parseObjectCandidate(candidate)
        if (parsed) {
          candidates.push({ candidate, parsed })
        }
        currentStart = -1
      }
    }
  }

  return candidates
}

function extractBalancedObjectFromStart(text, startIndex) {
  if (startIndex < 0 || text[startIndex] !== '{') {
    return null
  }

  let depth = 0
  let inString = false
  let escaped = false

  for (let index = startIndex; index < text.length; index += 1) {
    const char = text[index]

    if (inString) {
      if (escaped) {
        escaped = false
      } else if (char === '\\') {
        escaped = true
      } else if (char === '"') {
        inString = false
      }
      continue
    }

    if (char === '"') {
      inString = true
      continue
    }

    if (char === '{') {
      depth += 1
      continue
    }

    if (char === '}') {
      if (depth === 0) continue
      depth -= 1
      if (depth === 0) {
        return text.slice(startIndex, index + 1).trim()
      }
    }
  }

  return null
}

function extractJsonCandidate(raw) {
  const trimmed = String(raw || '').trim()
  if (!trimmed) return null

  const full = parseObjectCandidate(trimmed)
  if (full && looksLikeReviewPayload(full)) {
    return trimmed
  }

  const fenced = [...trimmed.matchAll(/```(?:json)?\s*([\s\S]*?)```/gi)]
  for (const match of fenced) {
    const candidate = match[1].trim()
    const parsed = parseObjectCandidate(candidate)
    if (parsed && looksLikeReviewPayload(parsed)) {
      return candidate
    }
  }

  const objects = collectJsonObjectCandidates(trimmed)
  for (let index = objects.length - 1; index >= 0; index -= 1) {
    if (looksLikeReviewPayload(objects[index].parsed)) {
      return objects[index].candidate
    }
  }

  for (let start = trimmed.lastIndexOf('{'); start !== -1; start = trimmed.lastIndexOf('{', start - 1)) {
    const candidate = extractBalancedObjectFromStart(trimmed, start)
    if (!candidate) continue
    const parsed = parseObjectCandidate(candidate)
    if (parsed && looksLikeReviewPayload(parsed)) {
      return candidate
    }
  }

  return null
}

module.exports = {
  extractJsonCandidate,
}

if (require.main === module) {
  const fs = require('fs')
  const input = fs.readFileSync(0, 'utf8')
  const candidate = extractJsonCandidate(input)
  if (!candidate) process.exit(1)
  process.stdout.write(candidate)
}
