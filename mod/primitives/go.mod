module github.com/berachain/beacon-kit/mod/primitives

go 1.22.3

replace github.com/berachain/beacon-kit/mod/errors => ../errors

require (
	github.com/berachain/beacon-kit/mod/errors v0.0.0-00010101000000-000000000000
	github.com/ethereum/go-ethereum v1.14.4-0.20240530142416-2262bf34158e
	github.com/ferranbt/fastssz v0.1.4-0.20240422063434-a4db75388da1
	github.com/golang-jwt/jwt/v5 v5.2.1
	github.com/holiman/uint256 v1.2.4
	github.com/minio/sha256-simd v1.0.1
	github.com/prysmaticlabs/gohashtree v0.0.4-beta
	github.com/stretchr/testify v1.9.0
	golang.org/x/sync v0.7.0
)

require (
	github.com/cockroachdb/errors v1.11.3 // indirect
	github.com/cockroachdb/logtags v0.0.0-20230118201751-21c54148d20b // indirect
	github.com/cockroachdb/redact v1.1.5 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/getsentry/sentry-go v0.28.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/klauspost/cpuid/v2 v2.2.7 // indirect
	github.com/kr/pretty v0.3.1 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/rogpeppe/go-internal v1.12.0 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	golang.org/x/crypto v0.23.0 // indirect
	golang.org/x/sys v0.20.0 // indirect
	golang.org/x/text v0.15.0 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
