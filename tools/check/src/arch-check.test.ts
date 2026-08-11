import { describe, expect, it } from 'bun:test'
import { parseArchitectureDoc } from './arch-check.ts'

describe('parseArchitectureDoc', () => {
  it('parses frontmatter and known prose sections', () => {
    const data = parseArchitectureDoc(`---
context: repo
updated_at: 2026-08-11
---

# Architecture

## Overview
the design

## Structure
the tree
`)
    expect(data.context).toBe('repo')
    expect(data.overview).toBe('the design')
    expect(data.structure).toBe('the tree')
  })
})
