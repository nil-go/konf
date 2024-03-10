module github.com/nil-go/konf/examples/azure

go 1.22

require (
	github.com/nil-go/konf v0.9.1
	github.com/nil-go/konf/provider/azappconfig v0.9.1
	github.com/nil-go/konf/provider/azblob v0.9.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/Azure/azure-sdk-for-go/sdk/azcore v1.10.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/azidentity v1.5.1 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/data/azappconfig v1.1.0 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/internal v1.5.2 // indirect
	github.com/Azure/azure-sdk-for-go/sdk/storage/azblob v1.3.1 // indirect
	github.com/AzureAD/microsoft-authentication-library-for-go v1.2.2 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/pkg/browser v0.0.0-20240102092130-5ac0b6a4141c // indirect
	github.com/stretchr/testify v1.9.0 // indirect
	golang.org/x/crypto v0.21.0 // indirect
	golang.org/x/net v0.22.0 // indirect
	golang.org/x/sys v0.18.0 // indirect
	golang.org/x/text v0.14.0 // indirect
)

replace (
	github.com/nil-go/konf => ../..
	github.com/nil-go/konf/provider/azappconfig => ../../provider/azappconfig
	github.com/nil-go/konf/provider/azblob => ../../provider/azblob
)
