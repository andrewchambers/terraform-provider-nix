package main

import (
	"os"

	"github.com/andrewchambers/terraform-provider-nix/nix"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceNixBuild() *schema.Resource {
	return &schema.Resource{
		Read: dataNixBuildRead,
		Schema: map[string]*schema.Schema{
			"nix_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"expression_path": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"store_path": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataNixBuildRead(d *schema.ResourceData, m interface{}) error {
	nixPath := os.Getenv("NIX_PATH")
	if p, ok := d.GetOk("nix_path"); ok {
		nixPath = p.(string)
	}

	expressionPath := d.Get("expression_path").(string)

	storePath, err := nix.BuildExpression(nixPath, expressionPath, nil)
	if err != nil {
		return err
	}

	id := d.Id()
	if id == "" {
		d.SetId(randomID())
	}

	err = d.Set("store_path", storePath)
	if err != nil {
		return err
	}

	return nil
}
