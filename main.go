// Package main novi_shop API.
//
//	@title			novi_shop API
//	@version		1.0
//	@description	REST API backend untuk aplikasi kasir/toko novi_shop.
//	@host			localhost:3030
//	@BasePath		/
//
//	@securityDefinitions.apikey	BearerAuth
//	@in								header
//	@name							Authorization
//	@description					Masukkan token dengan format: Bearer {access_token}
package main

import "shop_project_be/cmd"

func main() {
	cmd.Execute()
}
