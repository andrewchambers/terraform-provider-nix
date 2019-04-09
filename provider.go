package main

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/hashicorp/terraform/helper/schema"
)

// Provider creates the root nix terraform provider.
func Provider() *schema.Provider {
	return &schema.Provider{
		DataSourcesMap: map[string]*schema.Resource{
			"nix_build": dataSourceNix(),
		},
		ResourcesMap: map[string]*schema.Resource{
			"nix_nixos": resourceNixOS(),
		},
	}
}

func randomID() string {
	b := make([]byte, 32, 32)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}
	return hex.EncodeToString(b)
}
