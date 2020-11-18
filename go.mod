module github.com/juju/bundlechanges/v3

go 1.14

require (
	github.com/gobwas/glob v0.2.4-0.20181002190808-e7a84e9525fe // indirect
	github.com/juju/charm/v8 v8.0.0-20200908083540-3ea1a8c7a8df
	github.com/juju/charmrepo/v6 v6.0.0-20200817155725-120bd7a8b1ed
	github.com/juju/collections v0.0.0-20180717171555-9be91dc79b7c
	github.com/juju/errors v0.0.0-20200330140219-3fe23663418f
	github.com/juju/loggo v0.0.0-20190526231331-6e530bcce5d8
	github.com/juju/names/v4 v4.0.0-20200424054733-9a8294627524
	github.com/juju/naturalsort v0.0.0-20180423034842-5b81707e882b
	github.com/juju/testing v0.0.0-20191001232224-ce9dec17d28b
	github.com/juju/worker/v2 v2.0.0-20200916234526-d6e694f1c54a // indirect
	github.com/kr/pretty v0.2.1
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15
	gopkg.in/macaroon-bakery.v2 v2.1.1-0.20190613120608-6734dc66fe81 // indirect
	gopkg.in/yaml.v2 v2.2.7
)

replace gopkg.in/juju/worker.v1 => github.com/juju/worker/v2 v2.0.0-20200424114111-8c6ac8046912
