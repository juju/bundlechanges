module github.com/juju/bundlechanges/v3

go 1.14

require (
	github.com/gobwas/glob v0.2.4-0.20181002190808-e7a84e9525fe // indirect
	github.com/juju/charm/v8 v8.0.0-20201117030444-62c13a9fe0f0
	github.com/juju/charmrepo/v6 v6.0.0-20201118043529-e9fbdc1a746f
	github.com/juju/collections v0.0.0-20200605021417-0d0ec82b7271
	github.com/juju/errors v0.0.0-20200330140219-3fe23663418f
	github.com/juju/loggo v0.0.0-20200526014432-9ce3a2e09b5e
	github.com/juju/names/v4 v4.0.0-20200923012352-008effd8611b
	github.com/juju/naturalsort v0.0.0-20180423034842-5b81707e882b
	github.com/juju/testing v0.0.0-20200923013621-75df6121fbb0
	github.com/juju/worker/v2 v2.0.0-20200916234526-d6e694f1c54a // indirect
	github.com/kr/pretty v0.2.1
	gopkg.in/check.v1 v1.0.0-20200902074654-038fdea0a05b
	gopkg.in/juju/names.v3 v3.0.0-20191210002836-39289f373765 // indirect
	gopkg.in/macaroon-bakery.v2 v2.1.1-0.20190613120608-6734dc66fe81 // indirect
	gopkg.in/yaml.v2 v2.3.0
)

replace gopkg.in/juju/worker.v1 => github.com/juju/worker/v2 v2.0.0-20200424114111-8c6ac8046912
