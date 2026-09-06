# Changelog

## [0.1.0-alpha.3](https://github.com/tnware/freehand-stt/compare/v0.1.0-alpha.2...v0.1.0-alpha.3) (2026-09-06)


### Features

* add cleanup controls and require reasoning off for S1-mini ([#22](https://github.com/tnware/freehand-stt/issues/22)) ([348b2fe](https://github.com/tnware/freehand-stt/commit/348b2fe5e062e878dcec57208c608ff4c7bbee65))
* add compatibility profiles and site backend directory ([#20](https://github.com/tnware/freehand-stt/issues/20)) ([917dfd3](https://github.com/tnware/freehand-stt/commit/917dfd395d4a95dc998543146905d648bffab9b4))
* add explicit model profiles across speech features ([#30](https://github.com/tnware/freehand-stt/issues/30)) ([59aaf03](https://github.com/tnware/freehand-stt/commit/59aaf033df4958e510dd4ff602676a01f55e1587))
* add optional transcription context, hotwords, and temperature ([#21](https://github.com/tnware/freehand-stt/issues/21)) ([b36ce15](https://github.com/tnware/freehand-stt/commit/b36ce15a30f1b1240fe789933a855784d58c7e04))
* add transcription language selection and English-only S1-mini cleanup ([#26](https://github.com/tnware/freehand-stt/issues/26)) ([f16b96a](https://github.com/tnware/freehand-stt/commit/f16b96a41c123976b1dfd403ad1cb84c214f44b9))
* add whisper.cpp and vLLM provider compatibility ([#23](https://github.com/tnware/freehand-stt/issues/23)) ([fd2a870](https://github.com/tnware/freehand-stt/commit/fd2a87018f219ae36665ce9ddf7cd928deffa3d8))
* **connections:** add actionable metadata diagnostics ([#32](https://github.com/tnware/freehand-stt/issues/32)) ([a9ae42f](https://github.com/tnware/freehand-stt/commit/a9ae42f31ebacef72dc71811b23c123f7f3f737f))
* persist settings in SQLite with migration and recovery support ([#27](https://github.com/tnware/freehand-stt/issues/27)) ([0ec6ac3](https://github.com/tnware/freehand-stt/commit/0ec6ac3c799231498155466694eb50db8a637eb5))
* remember model settings per connection and feature ([#31](https://github.com/tnware/freehand-stt/issues/31)) ([eef9e36](https://github.com/tnware/freehand-stt/commit/eef9e36bb01c2b59bbf043cc10cd8ad239d84f9a))
* **settings:** add reusable connections and feature selections ([#28](https://github.com/tnware/freehand-stt/issues/28)) ([2577566](https://github.com/tnware/freehand-stt/commit/2577566f64293adf22ba8e9b361a3215266cf7fd))
* **setup:** add connections directly from task pickers ([#35](https://github.com/tnware/freehand-stt/issues/35)) ([e9ab884](https://github.com/tnware/freehand-stt/commit/e9ab88451f3cd24a2fa7cb108737b4d43de71dba))
* **tts:** add Kokoro support and provider voice discovery ([#33](https://github.com/tnware/freehand-stt/issues/33)) ([ecc5344](https://github.com/tnware/freehand-stt/commit/ecc53449c5ce2af3dcb092030131d5b70c2197e2))
* unify provider icons and clarify cost and hardware choices ([#25](https://github.com/tnware/freehand-stt/issues/25)) ([c5b97b4](https://github.com/tnware/freehand-stt/commit/c5b97b4c27c34879203264233292946aff691b99))


### Bug Fixes

* **app:** consolidate workflow state and runtime lifecycle ([#34](https://github.com/tnware/freehand-stt/issues/34)) ([d278f01](https://github.com/tnware/freehand-stt/commit/d278f01b5f81be9488f935204e112e0096290c1c))
* **lifecycle:** bound shutdown and prevent late playback ([#37](https://github.com/tnware/freehand-stt/issues/37)) ([bc70d80](https://github.com/tnware/freehand-stt/commit/bc70d806aeeb25c3b523c3c359c3a4c4d28068fd))
* prevent transcription replay and clarify connection checks ([#17](https://github.com/tnware/freehand-stt/issues/17)) ([e6fd6bc](https://github.com/tnware/freehand-stt/commit/e6fd6bc53c68722c7d3409a16d65969c33cc519c))
* reject truncated cleanup and clarify request contracts ([#19](https://github.com/tnware/freehand-stt/issues/19)) ([de70d5e](https://github.com/tnware/freehand-stt/commit/de70d5e4d94bdcec83f676ebd0487cf4f732b3a9))
* **theme:** align desktop dark mode with navy branding ([#29](https://github.com/tnware/freehand-stt/issues/29)) ([8855380](https://github.com/tnware/freehand-stt/commit/8855380cbfdd2491ef744676e712ae238b1c706e))

## [0.1.0-alpha.2](https://github.com/tnware/freehand-stt/compare/v0.1.0-alpha.1...v0.1.0-alpha.2) (2026-09-05)


### Features

* **site:** add particle background and refine section layouts ([#5](https://github.com/tnware/freehand-stt/issues/5)) ([4f02d11](https://github.com/tnware/freehand-stt/commit/4f02d11476ae490386dde040318b08451dd6202e))


### Bug Fixes

* **security:** block inference redirects and sanitize response metadata ([#9](https://github.com/tnware/freehand-stt/issues/9)) ([0cebb33](https://github.com/tnware/freehand-stt/commit/0cebb33b8b2ea3ec1c11bbadb6556b7e83bfc51a))

## 0.1.0-alpha.1 (2026-09-04)


### Features

* publish Freehand alpha ([9b58098](https://github.com/tnware/freehand-stt/commit/9b580981f0ce40e267bbf29b33df108782936c19))
