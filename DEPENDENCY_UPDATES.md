# Dependency maintenance — 2026-11-02

Updated the existing dependency graph within compatible version lines, including gRPC 1.85.0 and x/net 0.62.0. The build compiler floor is now Go 1.26.10 to use the patched standard library and HTTP/2 dependencies. Existing CI/release jobs already read the version from go.mod. This raises the source-build toolchain requirement, without changing Terraform resource schemas or the provider protocol.

Updated clavex-sdk-go from v0.3.0 to v0.4.0 after reviewing the tag diff: it adds DevicePKI support and does not alter existing service APIs. No provider resources were added. Race tests, go vet, go mod verify and govulncheck were run. Acceptance tests against a real Clavex installation were not run. No infrastructure or Terraform state was changed.
