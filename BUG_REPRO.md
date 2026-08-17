# BUG_REPRO

The following failures were observed while validating the initial project state.
Each section records what failed, how to reproduce it, and the complete command output.
They are preserved intentionally; only failing build gates are omitted from the generated Dockerfile.

## Failure 1: Go test (.)

- Observed problem: `Go test (.)` failed in the initial project state.
- Working directory: `.`
- Command: `cd /app && GOTOOLCHAIN=local GOPROXY=off GOSUMDB=off go test -count=1 ./...`
- Exit status: `1`

```text
?   	service-request-dispatch/cmd/server	[no test files]
ok  	service-request-dispatch/internal/audit	0.026s
ok  	service-request-dispatch/internal/config	0.007s
ok  	service-request-dispatch/internal/filter	0.007s
ok  	service-request-dispatch/internal/httpapi	0.235s
ok  	service-request-dispatch/internal/model	0.009s
ok  	service-request-dispatch/internal/queue	0.008s
ok  	service-request-dispatch/internal/routing	0.006s
--- FAIL: TestPartialFailureReportsErrorAndCleansRecord (0.09s)
    service_test.go:73: expected dispatch validation failure
FAIL
FAIL	service-request-dispatch/internal/service	0.281s
ok  	service-request-dispatch/internal/store	0.229s
FAIL
```

## Architecture reproduction

### linux/amd64
- Go toolchain version: exit `0`
- Go build (.): exit `0`
- Go test (.): exit `1`
- Go run smoke (cmd/server): exit `0`
### linux/arm64
- Go toolchain version: exit `0`
- Go build (.): exit `0`
- Go test (.): exit `1`
- Go run smoke (cmd/server): exit `0`
