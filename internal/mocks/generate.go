package mocks

//go:generate go run github.com/vektra/mockery/v2@v2.53.5 --name Repository --dir ../domain/league --output domain/league --outpkg leaguemock --filename repository_mock.go
//go:generate go run github.com/vektra/mockery/v2@v2.53.5 --name Repository --dir ../domain/team --output domain/team --outpkg teammock --filename repository_mock.go
//go:generate go run github.com/vektra/mockery/v2@v2.53.5 --name Repository --dir ../domain/fixture --output domain/fixture --outpkg fixturemock --filename repository_mock.go
