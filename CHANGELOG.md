# Changelog

## v0.5.0 (April 19, 2023)

### New Features

* Breaking changes:
    * Remove custom context from `Action` interface (8fce8d9)
    * Fix spelling of `Uint` to be idiomatic (dd17c3f)
    * Encapsulation of context (86a4836)
* `ArgCount` support for `*Arg`, `*Flag` (10eb0b8)
* Make `Transform` API (e4bfb43)
* Make `flag.Getter` provide the semantics of `dereference()` (c5dfaed)
* Expose `Flags`, `PersistentFlags`, `LocalFlags`, `LocalArgs` as API (09fbb9a)
* `SetData`; make setting transform results API (2b3ee10)
* Allow decoder options to `provider.Factory` (bcc7d70)
* Expose `ErrorUnused` as option to providers (bcc7d70)
* Introduce `provider.WithServices` using basic context type (2a631d8)

### Bug fixes and improvements

* Bug fixes: expression parsing (daa8921)
* Use `option` in binding instead of internals (ae3aa84)
* Simplification to triggering before/after options (5630833)
* Remove args from command context (23db2a2)
* Remove handling of names from internal option solely to binding (6615a07)
* Clean up internals and encapsulation of `internalOption` and remove `generic` (36575b8, a9587c9, 6a2674f)
* Optimization: remove unnecessary allocations (1dd9574)
* Addresses issues of style; update staticcheck (abadd68)
* Formalization/additional test coverage (64aa7f6)
* Cleanup: remove `exprContext` (2cf7b08)
* Chores:
    * GitHub CI update (4b92a47)
    * Goreleaser configuration (b5a99e4)


## v0.4.0 (March 19, 2023)

### New Features

* Introduce `RemoveFlag`, `RemoveCommand` and initialization internal state (d9796c4)
* Provider `Factory` reflection helper (5f347fe)
* Generalize provider `Lookup`; `Registry`.`New` (37d5c63)
* Structure: extract `Decode` function as API (afb0e11)
* Porcelain table format (bbabac9)
* Export `SplitMap` as API (812291b)
* `ByteLength`, `Hex`, `Octal` (30a8595)
* `FromContext` (63a17e6)
* `Alias` action (786c966)
* Breaking changes:
    * Drop `AtTiming` (2c789f7)

### Bug fixes and improvements

* Allow implicit conversion to primitives in lookup (8a62a20)
* Improvements to decoding maps (01c129c)
* Detect `COLUMNS` var; Relocate `guessWidth` to support (4e1affe)
* Chores:
    * Remove documentation from joe-cli repo (39db584)
    * Update engineering platform (c5fa6f8)
    * Update CI to go1.19 and go1.20 (874af1b)
    * Update dependent versions (1791d42)

## v0.3.1 (February 20, 2023)

### New Features

* Allow FS as argument to `FromFilePath` (e5dc8b3)
* Introduce `TransformFunc` as API (218c806)
* Allow `PrintVersion` to be used as an initializer; add -h alias (`4291c94`)
* Introduce `At` (to eventually replace `AtTiming`) (5218990)

### Bug fixes and improvements

* Updates to documentation (c30d8fc)
* Improve semantics of `SetData` to make it consistent on all targets (64127e5)
* Bug fix: Avoid possible panics if context FS is unset in some places (2da44ec)
* Bug fix: Treat explicitly set empty value as false when a flag is initialized via an environment variable  (755b4e0)
* Bug fix: Use primary flag synopsis in action group on the help screen (4291c94)
* Bug fix: copy `Aliases` to command in prototype (7756b81)
* Bug fix: check `HookBefore` used at correct timing (2de5a86)
* Bug fix: Ensure context value available in the proper context (2da4d23)
* Optimizations:
    * Refactor to move context pipeline behavior into global definitions (f1bd86b)
    * Set - use binding impl (3ea1e1f)
    * Remove closure from bubble, tunnel; from actions (ec9f71b)
    * Remove non-necessary code that simplifies test setup (cc80928)
    * Use decompose in `FeatureMap` to reduce code clones (147508d)
    * Remove unecessary deferred initialization in `hooksSupport` (748b3e3)
    * Remove redundant invocation of `Pipeline` (1d3fb24)
    * Change `copy(_, false)` to `copyWithoutReparent` (f7c7931)
    * Increase test coverage (ed0d25f)
* Address various issues of style (f9b1234)

## v0.3.0 (February 15, 2023)

### New Features
* Make `CurrentApp` function API and use sync atomic value (7828191)
* Introduce `Context.Use` (e7cb5bc)
* Introduce `Hook` to allow hooks at other timings (201472d)
* `FromEnv`, `FromFilePath` to extract logic of obtaining values from env and file path (fa837c4)
* Context filters to support custom predicates (9ea8f2f, a75cade)
* License support in joe (7791369)
* Add `PrintLicense` action, template (a499cf0)
* Add `NArg` to `Prototype`, supporting args (c846efd)
* Add support for prototype to `Command` (2da5267)
* Add `ManualText` action (41ab65f)
* Templates: `Touch`, allow nil file generators (6fe8207)

### Bug fixes and improvements

* Remove `exec.Args` API (81d53fc)
* Bug fixes: context paths (7ab1177)
* Bug fixes: `CurrentApp` not being cleared (7828191)
* Tests for exit (a6b0f66)
* Simplify `Accessory` to remove formatting logic (3e976ab)
* Refactor `ExecuteSubcommand` as prototype (ca73637)
* Add -h as default alias to help version (ed9a51c)
* Simplify hooks code by using pipelines (201472d)
* Bug fix: implies requires an occurrence (44984ab)
* Consolidate formatting logic into internal support package (01b7621)
* Fix Goreleaser description of `joe` (9f254c5)
* Decouple set and make `BindingLookup` API (2113636)
* Fix: Allow `File` to use `Synopsis` convention (cb03443)
* Chores:
    * Address linter errors (b3c1875)
    * Re-generate rad stubs (bc249b1)
    * Increase code coverage in exec extension (cde87e9)

## v0.2.2 (August 15, 2022)

### New Features

* Introduce `joe`, a command line utility for generating Joe-cli commands (998cdfa, d500da1)
* Template extension (10c790d)
* Add Emoji to color extension (a1c84e0)
* Introduce FS and update File to use it (b9a8232)
* Support `TextUnmarshaler` as flag automatically (5052364)
* Introduce `SetDescription`/`SetCategory` API on context (cb6a4ca)
* Make `Description` into `interface{}` (c4ea46b)
* Introduce template binding (0839520)
* Allow `CompletionFunc`, `StandardCompletion` as action (0dba5fd)
* Add `SpaceBefore`, `SpaceAfter` as template funcs (a4487d9)
* Support hidden commands (778b880)
* NameValue file references (cfadcad)

Introduction of a variety of new APIs:

* `CurrentApp` (c9650a9)
* `Implicitly` (c0c9ebf)
* `Mutex` (03379ec)
* `TakeExceptForFlags`  (9208241)
* `Use` (0196c9f)
* `SetCompletion` (0dba5fd)
* `ValueTransform` (89e9991)
* `TransformFileReference` (89e9991)

### Bug fixes and improvements

* Increase code coverage, fixes (e76166a, e916d04, 3f68a19)
* Improvements to color extension (edb078a)
* Improvements to transforms, `EachOccurrence` (89e9991)
* Additional tests for `FileReference`: multiple files (f0eb9d7)
* Bug fixes: allow string to be used with `MustExist` (ce324dc)
* Bug fix: call underlying `Context` indirectly when used as the param in action (41aba83)
* Factor internal flags out of expression (4b80709)
* Parsing and bind prototype bug fixes (8a0e2fa)
* Chores:
    * Collapse generated output in Makefile (117edd3)
    * Update engineering platform (656582d)
    * Fix build and CI on Windows (d3adbce, be4b4f1, d81e09f)
    * Linter, go mod tidy (7d1f186)
    * Adopt go1.19 formatting and builds (866fc0b, 293b8a8)
    * Update dependent versions (6b1bfa0)
    * Makefile: Introduce coveragereport, coverage (b75586b)


## v0.2.1 (July 9, 2022)

* Completion bug fixes (d04498d)
* Build fixes on Linux (15ced0b) and CI workflow (bbcc3ab)

## v0.2.0 (July 7, 2022)

### New Features

* Initial implementation of shell completion, supporting ZSH (c4c5f6d and 9c7aa52)
* Provides initial implementation of exec (399faff) and table (70aff80, 991e2ad) extensions
* Support `Bytes` from hex as a value (9560952)

Introduction of a variety of new APIs:

* `Accessory` (7a52416)
* `Bind`, `BindContext`, `BindIndirect` (0bceae9)
* `CompletionValues` (9c7aa52)
* `EachOccurrence` (6371979)
* `Enum` (83d19a6)
* `HandleCommandNotFound` (2ba8172)
* `ImplicitCommand` (cf54f74)
* `ImpliedAction`  (1d719df)
* `Implies` (8216a9e)
* `Prototype` (d0b6271)
* `Raw` (4d346cf)
* `RawParse` (f1baa7c)
* `ReadString` and `ReadPasswordString` (1237270)
* `Root` (59a5533)
* `Sorted` (82598c2)
* `Transform`  (9560952)
* `ValidatorFunc` (2a99dc1)

Examples:

* Introduce find, git examples (c3aa52c)

### Bug fixes and improvements

* Bug fix: allow blank names in `MustExists`, `WorkingDirectory` (52de5f4)
* Bug fix: detect short equal sign usages (19ae384)
* Bug fix: return template error when template is missing (11c9cba)
* Bug fix: nested prototypes should apply in order (1f505a3)
* Fix: standalone `--no-color` flag should have unique help text (99cedf5)
* Bug fix: middleware handling when added via `Uses` doesn't nest properly (85d6ca3)
* Bug fix: improve short parsing for flag-only flags (013121b)
* Bug fix: bind pointer semantics (d3c1dc7)
* Bug fix: `RawOccurrences` with an arg (17bcb53)
* Sub-timings for `Before` pipeline (1d719df)
* Make `Append` variadic like its counterpart (dc3ccab)
* Introduce `Arg.Category` (aa17730)
* Optional setup (3600852)
* Add `Comment` to app and command (9979239)
* Rework parsing and add `RawParse` API (f1baa7c)
* Color (5680116) and provider (237d092) extension improvements
* Replace `os.Exit` with panic for improperly registered flags (c1d8a31)
* Treat arg counter default differently between flag and arg (19a58a8)
* Allow `OpenContext` convention in FS (3f723c5)
* Provide `IsBoolFlag` convention for values per package flag (ad878eb)
* Unify `Execute` and `ExecuteSubcommand` (7841447)
* Remove appContext (59a5533) and consolidate option context (13a9e26)
* Improve tests and documentation for `Optional`/`OptionalValue` (cca5d7e)
* Earlier, definite initialization of flag set; remove RTL from internals (9e319c3)
* Remove redundant call to setTiming (92f6f37)
* Drop target conventions (dfcf10e)
* Bug fix: app from context should check (afbd103)
* Parsing bug fixes (547822c)
* Tag and source annotation in color extension (348773b)
* `RemoveArg`, plus generalization of arg by index in `Seen`, `Occurrences` (db4dc30)
* Various chores (bad4c37 and ffad000)
* Encapsulate panic data (6bb4e84)
* Bug fix: hide empty flag and command categories (2b31d74)
* Templates refactoring, global context (027e00f)
* Bug fix: Accessory should use conditional sprintf (e91aae6)

## v0.1.0 (May 1, 2022)

* Initial version :sunrise:
