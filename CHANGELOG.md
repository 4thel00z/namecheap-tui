# Changelog

## [0.2.0](https://github.com/4thel00z/namecheap-tui/compare/v0.1.0...v0.2.0) (2026-07-22)


### Features

* add account profile and credentials types ([7f3c250](https://github.com/4thel00z/namecheap-tui/commit/7f3c2504622608ad713f262df068c42a1c528e6a))
* add cobra root and profile commands ([691f589](https://github.com/4thel00z/namecheap-tui/commit/691f5894bda7a343e5c9813cdeee5b49b27fc11e))
* add config command, tabbed dashboard, and full README ([8e20d5b](https://github.com/4thel00z/namecheap-tui/commit/8e20d5b0040627615d18e3f264acc1be11bf97ef))
* add dns and ns command trees with yaml import/export ([654654e](https://github.com/4thel00z/namecheap-tui/commit/654654eced61f9ec1353c2069b00d5221eb53978))
* add dns and ns services with fresh-state apply ([71fa839](https://github.com/4thel00z/namecheap-tui/commit/71fa83925e99b9ccbbbc31e980155c326ddd0530))
* add dns domain package with zone changeset merge ([8c56335](https://github.com/4thel00z/namecheap-tui/commit/8c56335aa271d53e858d5fcb02a25c86b689b75b))
* add DNSAPI and NSAPI ports ([1171ce8](https://github.com/4thel00z/namecheap-tui/commit/1171ce882a78b7d25bad73fea5fd730743a3d21c))
* add domain service with cache-aside reads ([5431e21](https://github.com/4thel00z/namecheap-tui/commit/5431e21b2fbd8a829ad3fdc23145c0794f2e188c))
* add domains list/check/info commands ([26ddbe7](https://github.com/4thel00z/namecheap-tui/commit/26ddbe703f592038dd5998d270e4de248f87b056))
* add hexagon ports and typed API errors ([7771e08](https://github.com/4thel00z/namecheap-tui/commit/7771e08f78d2416e64bd1033919eda8b9c04cec9))
* add interactive zone editor with staged changes ([592ace4](https://github.com/4thel00z/namecheap-tui/commit/592ace4bbeccf4bf5ad9683fa5d039c0f3bb39dd))
* add lifecycle and transfer commands with contact wizard ([cf150da](https://github.com/4thel00z/namecheap-tui/commit/cf150da42782ea5c6bb3b47d9eb4f4741be9614a))
* add lifecycle service methods and transfer service ([0b57c73](https://github.com/4thel00z/namecheap-tui/commit/0b57c73333eff9b2c7d803183d6b0d0b51922f58))
* add namecheap XML client core with throttling and error mapping ([73dba3c](https://github.com/4thel00z/namecheap-tui/commit/73dba3ccd7ba4eb511a9538dfe9f0fffea7bf40f))
* add NCP_BASE_URL override for local and e2e API endpoints ([adb1513](https://github.com/4thel00z/namecheap-tui/commit/adb1513dffa266c0d2d59a9e511ce2d57a67dcbc))
* add profile service with env-override resolution ([39f7de1](https://github.com/4thel00z/namecheap-tui/commit/39f7de15dbcdb0debf595eec903c4e1b04c60c76))
* add public IP resolver with service fallback ([e0ea65a](https://github.com/4thel00z/namecheap-tui/commit/e0ea65a2a32487cd66e49e1a77d45d897608e9cb))
* add registrar domain types with DomainName parsing ([c20f779](https://github.com/4thel00z/namecheap-tui/commit/c20f779f91f99e76d356045f79412e8781a2a9fd))
* add registrar lifecycle and transfer API adapters ([16d4a7e](https://github.com/4thel00z/namecheap-tui/commit/16d4a7e081d3e3147dc4b77f81c52fe8f35fcaca))
* add registrar lifecycle types (contacts, registration, transfers) ([a1db6d2](https://github.com/4thel00z/namecheap-tui/commit/a1db6d21c016fc95e8abf80fc11ea86938aacc47))
* add ssl, privacy, account, and address commands with wiring ([7b51ace](https://github.com/4thel00z/namecheap-tui/commit/7b51acef857ae9fb0bc9215225374e52cabd5046))
* add ssl, privacy, and account API adapters ([f2aa1fb](https://github.com/4thel00z/namecheap-tui/commit/f2aa1fbe8b15666c84829502015b908a2fae1ed9))
* add ssl, privacy, and account services ([c5ab4fe](https://github.com/4thel00z/namecheap-tui/commit/c5ab4fe215e9a46378371b3556d304a01313e524))
* add ssl, privacy, and account types with ports ([11bc905](https://github.com/4thel00z/namecheap-tui/commit/11bc9051bcee198fdbdf2cc9651e67ba62c43aa0))
* add TUI dashboard with stale-while-revalidate domain table ([e774500](https://github.com/4thel00z/namecheap-tui/commit/e774500ca05665355ba6fbbfefd9a0c87272adb3))
* add turso settings and cache repositories ([0750c0a](https://github.com/4thel00z/namecheap-tui/commit/0750c0a191db2318d5ed2c053601d9117d3cf545))
* add turso store with migrations and profile repository ([3aedaef](https://github.com/4thel00z/namecheap-tui/commit/3aedaef491e5e900b18d0330aa313f2ad681661d))
* bootstrap ncp module, version package, and Makefile ([fa421f5](https://github.com/4thel00z/namecheap-tui/commit/fa421f5382d47bdfede291d98264847ca6076771))
* implement DNSAPI and NSAPI on namecheap client ([e9bd223](https://github.com/4thel00z/namecheap-tui/commit/e9bd223749244dcd432bf714d36b84b95743c2f1))
* implement RegistrarAPI on namecheap client (list, check, info) ([b7c0e19](https://github.com/4thel00z/namecheap-tui/commit/b7c0e19b71c8fdceacb7ea0926865e9244218385))
* phase 1 foundation — hexagon, turso store, namecheap client, CLI, TUI dashboard ([15164c6](https://github.com/4thel00z/namecheap-tui/commit/15164c674f4c80c851a6f07050278a89e3927a10))
* phase 2 DNS — zone changesets, dns/ns commands, zone editor TUI ([e7cd62e](https://github.com/4thel00z/namecheap-tui/commit/e7cd62e7a0f4fe48e90220a5de375b4433194922))
* phase 3 registrar lifecycle — register, renew, contacts, lock, transfers ([64d44e5](https://github.com/4thel00z/namecheap-tui/commit/64d44e5add95780752d93dab9abeeb05220202c5))
* phase 4 — ssl certificates, domain privacy, balances, pricing, address book ([472d9cd](https://github.com/4thel00z/namecheap-tui/commit/472d9cd44dedb95d40e63d7b9119528cefe97b91))
* phase 5 polish — config sync, tabbed dashboard, docs ([ae0efd0](https://github.com/4thel00z/namecheap-tui/commit/ae0efd0708d2cb13ede3a95ba72ec3c7163aa57f))
* wire composition root with fang, turso, and TUI dashboard ([34c8aac](https://github.com/4thel00z/namecheap-tui/commit/34c8aac8a2f1dafc7ab5af87251035af43e5e93d))
* wire dns, ns, and zone editor into composition root ([65609a0](https://github.com/4thel00z/namecheap-tui/commit/65609a0285801b10ced927b91c7b5a8b66d66d0e))


### Bug Fixes

* check Fprint errors flagged by errcheck ([a7c5978](https://github.com/4thel00z/namecheap-tui/commit/a7c5978e2249dee6cfa0f23719936c8e6024e41d))
* render styled TUI table cells correctly ([740626e](https://github.com/4thel00z/namecheap-tui/commit/740626e5002b2a66abeb03b957ca4628121d24dc))

## Changelog
