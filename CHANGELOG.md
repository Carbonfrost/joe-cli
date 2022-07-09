# Changelog

## v0.2.1 (July 9, 2022)

* Completion bug fixes (d04498d)
* Build fixes on Linux (15ced0b) and CI workflow (bbcc3ab)

## v0.2.0 (July 7, 2022)

New Features

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

Bug fixes and improvements:

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
