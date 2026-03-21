---
created: 2026-03-21
title: Fix error swallowing in Execute()
area: cli
files:
  - cmd/root.go:34-38
---

## Problem

`SilenceErrors: true` on the root cobra command combined with no explicit error print in `Execute()` means users never see error messages when a command fails. The process exits with code 1 but no diagnostic output.

## Solution

Add `fmt.Fprintln(os.Stderr, err)` before `os.Exit(1)` in `Execute()`, or remove `SilenceErrors: true` and let cobra handle it. The first option is preferred since it keeps cobra's usage output suppressed (via `SilenceUsage: true`) while still showing the actual error.
