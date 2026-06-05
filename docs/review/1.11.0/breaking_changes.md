# go-unifi 1.11.0 — API breaking changes

This document tracks every public-API behavior or signature change introduced while implementing the
[1.11.0 review](summary.md). Each entry links to the finding ID that motivated it and the migration
guidance for downstream consumers.

> Status: populated wave by wave during implementation. Empty sections mean no breaking change landed
> in that wave (yet).

## Wave 0 — P0 hotfixes

_No breaking changes._ All three P0 fixes (ARCH-01 deadlock, ARCH-02 permissive `booleanishString`
decode, ARCH-03 missing setting factories) are bug fixes that only make previously-broken paths work;
no public signature or documented behavior changes.

## Wave 1 — P1 hardening

_To be populated._

## Wave 2 — P2 quality & codegen robustness

_To be populated._
