---
created: 2026-03-21
title: Extract shared pipeline stages into helper
area: pipeline
files:
  - internal/pipeline/pipeline.go:84-117
  - internal/pipeline/pipeline.go:184-217
  - internal/pipeline/pipeline.go:288-321
---

## Problem

Stages 4-7 (Sort, ValidateDAG, Emit, Write) are copy-pasted verbatim across `RunK8s`, `RunCRD`, and `RunProvider`. Any bug fix or stage change must be applied three times. This violates DRY and increases maintenance risk.

## Solution

Extract a shared helper function:

```go
func emitAndWrite(fileMap organizer.FileMap, pkg, outputDir string) (emitter.EmitResult, int, int, []string, error) {
    // sort, validate DAG, emit, write
}
```

Each pipeline function handles its unique stages (load, resolve, annotate, organize) then delegates to the shared helper for the common tail.
