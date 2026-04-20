# Changelog

## [0.4.1](https://github.com/Wompipomp/function-starlark-gen/compare/v0.4.0...v0.4.1) (2026-04-20)


### Bug Fixes

* omit enum kwarg for list fields to prevent invalid schema validation ([5b40f08](https://github.com/Wompipomp/function-starlark-gen/commit/5b40f081c235e671941cec67656cb4c588be10ec))

## [0.4.0](https://github.com/Wompipomp/function-starlark-gen/compare/v0.3.2...v0.4.0) (2026-04-19)


### Features

* default apiVersion/kind on top-level resource schemas ([b6fc15b](https://github.com/Wompipomp/function-starlark-gen/commit/b6fc15b8784272d3105ccf16f163dd054c98f3ee))

## [0.3.2](https://github.com/Wompipomp/function-starlark-gen/compare/v0.3.1...v0.3.2) (2026-03-21)


### Bug Fixes

* handle circular type dependencies in CRD schemas ([4de37b2](https://github.com/Wompipomp/function-starlark-gen/commit/4de37b26ffec8fe6f5c138379abe7961180c563e))

## [0.3.1](https://github.com/Wompipomp/function-starlark-gen/compare/v0.3.0...v0.3.1) (2026-03-21)


### Bug Fixes

* correct upload-artifact SHA pin in release workflow ([dbcca89](https://github.com/Wompipomp/function-starlark-gen/commit/dbcca89fb1b69739fd7ae4323827c07af8f3fb8a))

## [0.3.0](https://github.com/Wompipomp/function-starlark-gen/compare/v0.2.0...v0.3.0) (2026-03-21)


### Features

* **01-01:** implement Swagger 2.0 loader with libopenapi V2 model ([e5ef8d4](https://github.com/Wompipomp/function-starlark-gen/commit/e5ef8d4283840cd3c43e09d000f2131f6a86c05a))
* **01-01:** initialize Go module, define TypeNode/FieldNode types, create testdata ([2f8b324](https://github.com/Wompipomp/function-starlark-gen/commit/2f8b324c10d8f182b0f2f72a90428fedae15a17c))
* **01-02:** implement core resolver with ref resolution, circular refs, allOf, oneOf/anyOf, additionalProperties ([1abadde](https://github.com/Wompipomp/function-starlark-gen/commit/1abadde9ee5eb1f8bdf24c849cf744ae11d75c27))
* **01-02:** implement K8s extension handling and special type detection ([9fa1563](https://github.com/Wompipomp/function-starlark-gen/commit/9fa15634eafa66cd521911e1932727309280d04b))
* **01-03:** implement definition path mapping and file assignment organizer ([3a14f8c](https://github.com/Wompipomp/function-starlark-gen/commit/3a14f8c8be94bb6137211b2385038df72c6bbd10))
* **01-03:** implement TypeGraph topological sort and load DAG validation ([d96e399](https://github.com/Wompipomp/function-starlark-gen/commit/d96e39915dd5c573a54225d1e1f9ccd597fc414f))
* **01-04:** implement deterministic file writer with directory creation and schema counting ([3b0230a](https://github.com/Wompipomp/function-starlark-gen/commit/3b0230ab0507c534fd3c29b4a15c6c03a4db70b0))
* **01-04:** implement Starlark emitter with schema()/field() code generation ([161bfe3](https://github.com/Wompipomp/function-starlark-gen/commit/161bfe38630437e5ae05051822ca7d64a3e9d0ef))
* **01-05:** implement CLI with cobra root command and k8s subcommand ([922fca6](https://github.com/Wompipomp/function-starlark-gen/commit/922fca63618a497e3060ef556d9ec51d19afd44a))
* **01-05:** implement pipeline orchestrator wiring all five stages ([aee66ec](https://github.com/Wompipomp/function-starlark-gen/commit/aee66ec212ed75ff1b95d4c25c64aad08bbda22f))
* **02-01:** implement CRD resolver and path mapper ([6a2363a](https://github.com/Wompipomp/function-starlark-gen/commit/6a2363a2bb5e6c1c453c51d5792e1338e99b2295))
* **02-01:** implement CRD YAML loader with v1/v1beta1 and multi-doc support ([39ca059](https://github.com/Wompipomp/function-starlark-gen/commit/39ca059ff1144c9c8246aca02e14588d98a4487b))
* **02-02:** implement enum constants and default value emission ([ea48c39](https://github.com/Wompipomp/function-starlark-gen/commit/ea48c398c5a32727beb8dfefcc8b93ca7b840edc))
* **02-03:** implement crd cobra subcommand and register in root ([34c2c61](https://github.com/Wompipomp/function-starlark-gen/commit/34c2c61badf48fb3c6e3e76e6a06714e24144838))
* **02-03:** implement RunCRD pipeline function ([6a19d2c](https://github.com/Wompipomp/function-starlark-gen/commit/6a19d2c1e8c4bfdc8442bcb3f51c8141855e551c))
* **03-01:** implement AnnotateCrossplane with status removal and doc annotations ([348a121](https://github.com/Wompipomp/function-starlark-gen/commit/348a1211786a15b3887db4e6afbc3b51d8709edd))
* **03-02:** add provider cobra subcommand and register in root ([a419372](https://github.com/Wompipomp/function-starlark-gen/commit/a419372cbea36c04890be006859ac0fb2087f62e))
* **03-02:** implement RunProvider pipeline with Crossplane annotator ([0e85fe3](https://github.com/Wompipomp/function-starlark-gen/commit/0e85fe3834288cc628932263c75bddc7ca3b8878))
* **04-01:** add Starlark test harness with function-starlark schema dependency ([ad12dc6](https://github.com/Wompipomp/function-starlark-gen/commit/ad12dc60d383e281c8d398aa09e6ee68b4fadc3b))
* **04-02:** add example CI workflows for K8s and provider schema updates ([ad8fd75](https://github.com/Wompipomp/function-starlark-gen/commit/ad8fd750c69e4c489d26e7a642744494defb5cd8))
* add README and cross-platform release binaries ([5382ace](https://github.com/Wompipomp/function-starlark-gen/commit/5382ace2ece83b387f0dbf43c5e1aa7eab319376))


### Bug Fixes

* **04-01:** sanitize hyphenated field names in emitted Starlark code ([3fffe0e](https://github.com/Wompipomp/function-starlark-gen/commit/3fffe0e4d74fbb79a71f9243da07778e04846aee))
* **04-01:** use non-deprecated ExecFileOptions and remove unused findCallable ([fb7144d](https://github.com/Wompipomp/function-starlark-gen/commit/fb7144d15a14678e1e61714a63999bf94d788f7a))
* address code review findings ([fb97fa0](https://github.com/Wompipomp/function-starlark-gen/commit/fb97fa051b44b35850f1051804b351cf5e347f6d))

## [0.2.0](https://github.com/Wompipomp/function-starlark-gen/compare/v0.1.0...v0.2.0) (2026-03-21)


### Features

* **01-01:** implement Swagger 2.0 loader with libopenapi V2 model ([e5ef8d4](https://github.com/Wompipomp/function-starlark-gen/commit/e5ef8d4283840cd3c43e09d000f2131f6a86c05a))
* **01-01:** initialize Go module, define TypeNode/FieldNode types, create testdata ([2f8b324](https://github.com/Wompipomp/function-starlark-gen/commit/2f8b324c10d8f182b0f2f72a90428fedae15a17c))
* **01-02:** implement core resolver with ref resolution, circular refs, allOf, oneOf/anyOf, additionalProperties ([1abadde](https://github.com/Wompipomp/function-starlark-gen/commit/1abadde9ee5eb1f8bdf24c849cf744ae11d75c27))
* **01-02:** implement K8s extension handling and special type detection ([9fa1563](https://github.com/Wompipomp/function-starlark-gen/commit/9fa15634eafa66cd521911e1932727309280d04b))
* **01-03:** implement definition path mapping and file assignment organizer ([3a14f8c](https://github.com/Wompipomp/function-starlark-gen/commit/3a14f8c8be94bb6137211b2385038df72c6bbd10))
* **01-03:** implement TypeGraph topological sort and load DAG validation ([d96e399](https://github.com/Wompipomp/function-starlark-gen/commit/d96e39915dd5c573a54225d1e1f9ccd597fc414f))
* **01-04:** implement deterministic file writer with directory creation and schema counting ([3b0230a](https://github.com/Wompipomp/function-starlark-gen/commit/3b0230ab0507c534fd3c29b4a15c6c03a4db70b0))
* **01-04:** implement Starlark emitter with schema()/field() code generation ([161bfe3](https://github.com/Wompipomp/function-starlark-gen/commit/161bfe38630437e5ae05051822ca7d64a3e9d0ef))
* **01-05:** implement CLI with cobra root command and k8s subcommand ([922fca6](https://github.com/Wompipomp/function-starlark-gen/commit/922fca63618a497e3060ef556d9ec51d19afd44a))
* **01-05:** implement pipeline orchestrator wiring all five stages ([aee66ec](https://github.com/Wompipomp/function-starlark-gen/commit/aee66ec212ed75ff1b95d4c25c64aad08bbda22f))
* **02-01:** implement CRD resolver and path mapper ([6a2363a](https://github.com/Wompipomp/function-starlark-gen/commit/6a2363a2bb5e6c1c453c51d5792e1338e99b2295))
* **02-01:** implement CRD YAML loader with v1/v1beta1 and multi-doc support ([39ca059](https://github.com/Wompipomp/function-starlark-gen/commit/39ca059ff1144c9c8246aca02e14588d98a4487b))
* **02-02:** implement enum constants and default value emission ([ea48c39](https://github.com/Wompipomp/function-starlark-gen/commit/ea48c398c5a32727beb8dfefcc8b93ca7b840edc))
* **02-03:** implement crd cobra subcommand and register in root ([34c2c61](https://github.com/Wompipomp/function-starlark-gen/commit/34c2c61badf48fb3c6e3e76e6a06714e24144838))
* **02-03:** implement RunCRD pipeline function ([6a19d2c](https://github.com/Wompipomp/function-starlark-gen/commit/6a19d2c1e8c4bfdc8442bcb3f51c8141855e551c))
* **03-01:** implement AnnotateCrossplane with status removal and doc annotations ([348a121](https://github.com/Wompipomp/function-starlark-gen/commit/348a1211786a15b3887db4e6afbc3b51d8709edd))
* **03-02:** add provider cobra subcommand and register in root ([a419372](https://github.com/Wompipomp/function-starlark-gen/commit/a419372cbea36c04890be006859ac0fb2087f62e))
* **03-02:** implement RunProvider pipeline with Crossplane annotator ([0e85fe3](https://github.com/Wompipomp/function-starlark-gen/commit/0e85fe3834288cc628932263c75bddc7ca3b8878))
* **04-01:** add Starlark test harness with function-starlark schema dependency ([ad12dc6](https://github.com/Wompipomp/function-starlark-gen/commit/ad12dc60d383e281c8d398aa09e6ee68b4fadc3b))
* **04-02:** add example CI workflows for K8s and provider schema updates ([ad8fd75](https://github.com/Wompipomp/function-starlark-gen/commit/ad8fd750c69e4c489d26e7a642744494defb5cd8))


### Bug Fixes

* **04-01:** sanitize hyphenated field names in emitted Starlark code ([3fffe0e](https://github.com/Wompipomp/function-starlark-gen/commit/3fffe0e4d74fbb79a71f9243da07778e04846aee))
* **04-01:** use non-deprecated ExecFileOptions and remove unused findCallable ([fb7144d](https://github.com/Wompipomp/function-starlark-gen/commit/fb7144d15a14678e1e61714a63999bf94d788f7a))
* address code review findings ([e5d3898](https://github.com/Wompipomp/function-starlark-gen/commit/e5d38982b38e8ecc1e9a54560ae4204e11a1c9ac))
