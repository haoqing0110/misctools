module github.com/creachadair/misctools

go 1.13

require (
	bitbucket.org/creachadair/stringset v0.0.9
	github.com/creachadair/badgerstore v0.0.8
	github.com/creachadair/boltstore v0.0.0-20201108194349-10e56cb7e706
	github.com/creachadair/command v0.0.0-20200910004628-e48505ecfece
	github.com/creachadair/ffs v0.0.0-20201108195135-f943ac6b6670
	github.com/creachadair/gcsstore v0.0.0-20201108194514-7100a1a9d112
	github.com/creachadair/getpass v0.1.1
	github.com/creachadair/jrpc2 v0.10.5
	github.com/creachadair/keyfile v0.5.3
	github.com/creachadair/sqlitestore v0.0.0-20201108195105-defdf28c0304
	github.com/creachadair/vql v0.0.19
	github.com/tdewolff/minify/v2 v2.9.7
	golang.org/x/crypto v0.0.0-20201016220609-9e8e0b390897
	golang.org/x/net v0.0.0-20201109172640-a11eb1b685be // indirect
	golang.org/x/sys v0.0.0-20201109165425-215b40eba54c // indirect
)

replace github.com/creachadair/sqlitestore => ../sqlitestore
