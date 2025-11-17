# Changelog

## v0.9.3 (November 16, 2025)

### New Features

* `bind.SetPointer` (a5ace84)
* Handle fallback from unknown flags (2317fb8)
* `HasSeen` (c80ea55)

### Bug fixes and improvements

* Bug fix: allow JSON wrapper to ignore EOF (d7d48ae)
* Add source annotations to built-in version/help flags (9a15594)
* JSON wrapper support for copying and resetting inner value (d7d48ae)
* Convert args only into raw flag (c7e0076)
* Cleanup in `File`, `FileSet` (20f8a68)
* Chores:
    * Update GitHub configuration (9e5d188, ba50254, 5130bf1)
    * Update dependent versions (8fc0631, d17ed24)

## v0.9.2 (July 13, 2025)

### New Features

* `ActionOf`, add support for `error` as argument (9a3a7b4)
* Allow typed value dereferences (27e1af9)
* Bind extension:
    * Add `FileBinder` and delegates, allowing easier use of file components as bindings (242d3a8)
* Expression extension:
    * `EvaluatorOf`: Support additional function signatures (d370117)
    * `ComposeEvaluator` (78ba84b)

### Bug fixes and improvements

* Bug: Ensure `OptionalArg` is treated as optional in synopsis (f1a9a33)
* Chores:
    * Update to go1.24 (8a5d3cd)
    * Adopt coverralls (2ffc8b9)
    * Licensing and copyright information (6895338)
    * Adopt a code of conduct (8c1a09d)
    * Lint: address spelling errors; modernizations (e1549e9)
    * Address minor linter errors (95cf25a)
    * Remove a redundant call to `FromContext` (95cf25a)
* Adopt debug symbol info to control version in `joe` (a9743f2)
* Bug fix: ensure name value arg counter doesn't consume possible flags (c3f0a08)


## v0.9.1 (April 14, 2025)

### New Features

* Expr extension:
    * `Error` and additional signatures for `EvaluatorOf` (e143248)
    * `Predicate` and `Invariant` as types (cceef77)
* Breaking changes:
    * `TransformFileReference` change API to remove Boolean (`TransformOptionalFileReference`) (7eb6714)
* `FileSet` support for loading pathspec format files as input (3272b92)

### Bug fixes and improvements

* Bug fix: Ensure value splitting within `provider.Value` maps (04aeb81)
* Bug fix: Re-initialize sub-commands on rename (94ec6e1)
* Bug fixes: Usage template typos; visibility (2814c5b)
* Bug fix: Fix parse error to report all remaining (7f7d241)
* Validate names used in expression operators (e15dc20)
* Improve handling of ANSI colors on usage screen (5b01e62)
* Expr extension: remove need for `*Context` where unneeded (211bd0d)
* Rename `TokenCompletionType`, etc. to `CompletionTypeToken` for idiomaticness (ec69ae7)
* Chores:
    * Update dependent versions (d209b87)
    * Apply Go modernizations (4de634a, f07f29e)
    * Unit tests: Instead of `context.TODO` use `context.Background` (08cb8d2)


## v0.9.0 (March 17, 2025)

### New Features

* Introduces new APIs:
    * `InternalError` (d4254ed)
    * `Bindings` method on `Expression` (4751ed8)
    * `AddExpr`, `AddExprs` (43a49cd)
    * `SetEvaluator` (9a0c926)
    * `Trigger` (d950146)
    * `LookupValueTarget` (153f4a6)
    * `LookupFunc` (e7b2716)
    * Introduce `BindingMap.Apply` as API (d79a1f4)
    * Bind extension: `Redirect` (b12e771)
* Allow setting local args by convention (df0b5ed)
* Breaking changes:
    * Relocate expressions into own extension, `extensions/expr` (7a8a67f)
    * Relocate specialized value types into own package `value` (fd20f64)
    * Improve API consistency in use of `Action` and pipelines (2c20a0a)

### Bug fixes and improvements

* Bug fix: ensure `Customize` works with built-in version, help flags (4b2b3e5)
* Bug fix: `No` flag double invocation of before, after pipelines (11249b7)
* Bug: Fix cycles in context signals/channels leading to stackoverflow (a5f887d)
* Support visibility of expressions (ee4fc44)
* Detect internal errors:
    * Check valid identifiers in flag names (cdae336)
    * Detect duplicate names within args, flags, exprs (30b28e7)
    * Adds defense for invalid types for values (0d14827)
* Update `Fprint` methods to support `nil` with special semantics (2fe2e11)
* Add support for `Prototype` to value initializers (9747a08)
* Enable hook support on flags and args (8a8b162)
* Improvements to usage template (16832f2, b212b02, 596c077, 956d73d)
* Refactor to `DisplayHelpScreen`; bug fixes (4a675fb)
* Bind extension:
    * Allow more general naming of arguments (b12e771)
    * Allow later pipelines to use implicit name binds (3d567bf)
* Various generalizations, which are enablers for `expr` extension:
    * Generalization of expression bindings, binding map (0a6663f)
    * Generalization of sorted usage for expressions (6e4e87f)
    * Generalize set to handle Binding as API (c4e8114)
* Various cleanups and simplifications of internal implementation:
    * Terser implementation of Expression (9170b4d)
    * Refactor code clones: `EachOccurrence`, bind extension (3529317, b12e771)
    * Remove `internalCommandContext` (07a22f2)
    * Remove internal option (e54f867, a9908e0)
    * Rework updating args, flags, commands methods (1b80778)
    * Cleanup internal context lookup API (a5b64ae)
    * Move hooks actions into default command action pipelines (4a99dfc)
    * Push up methods on BindingMap from set (3a554da)
    * Remove `FromContext` calls where not necessary (387cdf6)
    * Remove internal context event methods (3c4ea11)
    * Unify methods that create child contexts (c593fcf)
* Chores:
    * Use base library `fstest` where possible (90b418a)
    * Addresses some linter errors (10dbd50)
    * Lint; modern configuration for tools (9d0aeb0)
    * Addresses documentation and lint errors (e0b284a)


## v0.8.0 (March 1, 2025)

### New Features

* `Named` (e18a8f9)
* `ValueContextOf` (845f5ac)
* `Fprint` actions (2c22b35)
* `Accessory0` action, a simpler version to support accessories (5d6ee2a)
* Expose `ColorCapable` as API (e1e696f)
* Bind extension: Indirect, Exact, Value (cf95843, b369930)
* Breaking changes:
    * Improve the signature of `Accessory` so that it is contravariant (5d6ee2a)

### Bug fixes and improvements

* Improve `Args` method by panicking sooner with invalid args (c19c091)
* Make `ProvideValueInitializer` implicitly initialize args and flags (5920087)
* Bug fix: License template and its generation from joe (c557a93)
* Bug fix: `ContextOf` should resolve from flag, arg contexts (34c4a22)
* Bug fix: avoid panics and incorrect results in `Lookup{Arg,Flag,Command}` when nil or out of range index is used (b95cd89)
* Convert `Synopsis` templates and logic into ordinary string manipulation (a463b1a)
* Ensure typing of color constants (41d9993)
* Additional unit tests and documentation improvements (43e01ac, 73ad697, fc006b2)
* Chores:
    * Use base library fstest where possible (90b418a)
    * Cleanup to remove some spurious calls or generalize (7943f6d)

## v0.7.0 (February 17, 2025)

### New Features

* Introduce the `bind` extension to facilitate common idioms (a1a6799)
* Introduce `ContextOf` (72cd831)
* Introduce `SortedExprs` (a50971c)
* Introduce `HasValue` filter modes (80fbbcf)
* Introduce `Assert` action (a4bcb98)
* Introduce `iter` support to `FileSet` (cdc4dc5)
* Add marshaling, describe to `FilterModes` and `Timing` (b8e7455)
* Support implicit and explicit visibility of flags and commands (e8d774c)
* Introduce `Context.Matches` method to increase symmetry between actions and context methods (144f09c)
* Allow `Target` to support for values in value initialization contexts (80e5bc0)
* Breaking changes:
    * Merge `CompletionContext` into `Context` (286f328)
    * Replace return value in `Initialize` with `context.Context` (5cba6a6)
    * Rename `Flag.{Short,Long}Name()` (3eab579)

### Bug fixes and improvements

* Split `lookupSupport` logic from `Context` (e936eee)
* Documentation fixes (be48a05)
* Various modernizations and cleanup (d290ad2, b1d7e2b, 046293d, acef537, a50971c, ef92c89)
* Remove internal state from annotations to internal flags (da285dd)
* Improve consistency in the definition of hooks; additional tests (90bdd8d)

## v0.6.0 (January 20, 2025)

### New Features

* Introduce `RemoveAlias` (f07d145)
* Introduce `Aliases` function (86aac47)
* Introduce `Print` and `Printf` (9bc2601)
* Introduce `UsageText`, `HelpText` actions (ed33e49)
* Shell detection improvements (e0c9bab)
* Improve consistency of action/context API by providing `Action` accessors where a similar `Context method exists (bef39bd)
* Support marshaling values as JSON using the `JSON` wrapper (4c51ed8)
* Breaking changes:
    * Allow `Quote` to be invoked with an untyped argument (4b5aaff)

### Bug fixes and improvements

* Improve safety of `exec.ArgList` API (fc6f9e6)
* Rename `RenderTemplate` to `ExecuteTemplate` (e1a9bef)
* Improvements to test coverage; test fixes (acd004b, 86aac47, 36c5da0, b33cdfa)
* Various code cleanups and documentation fixes (7d1e8d3, 767bc48, d0b508f)
* Bug fix: Detect `RawOccurrences` with `Before` on flags with zero occurrences (eead11b)
* Template extension refactoring (3769777)
* Chores:
    * Upgrade to go1.23 (2c50f72)
        * Take advantage of newer features such as `maps.Copy` (75cf239)
    * Upgrade dependent versions (a625b8b, 74c170c, 50ca385, cf5892a, f5f674d, 403ccf5, 800c9c4, e4ccb54, 61fe300, 5e7b0a5)
    * Completion: split out ZSH to own file (fb41688)

## v0.5.2 (May 4, 2023)

### New Features

* Provider extension help text; `ListProviders` improvements (b2ae12f)
* Export and document `ValueReader` (10f001f)

### Bug fixes and improvements

* Optimization: remove unnecessary syncing of internal option state (d750276)
* Bug fix: Ensure that default App Name propogates (9b8642f)
* Optimization: remove unneeded At invocation and use justBeforeTiming directly (4d408b2)
* Provider extension: Make list provider template conditional (88af848)
* Chores:
    * Fix CI configuration to go1.20 only (7a30ff0)
    * GitHub configuration: Dependabot (f6745b6)
    * Addresses issues of style to please linter (9ce729c)

## v0.5.1 (April 24, 2023)

### New Features

* Make `SetValue` untyped, allowing directly setting (b2b13eb)
* `Data`: get the data map from the context (218cc17)
* Expose Binding and binding names from context and binding lookup (77a6938)
* Expose `Flag` short and long name API (9ef19b6)
* Allow resetting `HandleCommandNotFound` and inheritance (5d68328)
* Expose `OpenContext` from FS API (c604cc4)
* Implicit value error, `ErrImplicitValueAlreadySet` (1087d73)
* Context API improvements; `Do` (36a1795)

### Bug fixes and improvements

* Allow simple action func in plain usages (1153507)
* Bug fix: Ensure `Context` is unwrapped in provider context services (edc2fcc)
* Allow legacy `Action` to be used in `ActionOf` (d250d94)
* Allow `[]byte` to have optional values (b08525b)

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
