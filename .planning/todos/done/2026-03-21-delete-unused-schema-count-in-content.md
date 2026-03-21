---
created: 2026-03-21
title: Delete unused schemaCountInContent function
area: cli
files:
  - cmd/k8s.go:89-91
---

## Problem

`schemaCountInContent` in `cmd/k8s.go` is dead code. `printVerboseOutput` uses `bytes.Count` directly at line 83 instead. The function also has an unnecessary `string()` allocation compared to the `bytes.Count` pattern used elsewhere.

## Solution

Delete the function. No callers exist.
