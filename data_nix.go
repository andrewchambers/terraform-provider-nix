package main

import (
	"github.com/andrewchambers/terraform-provider-nix/nix"
	"github.com/hashicorp/terraform/helper/schema"
)

func dataSourceNix() *schema.Resource {
	return &schema.Resource{
		Read: resourceNixRead,
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

func resourceNixRead(d *schema.ResourceData, m interface{}) error {
	var nixPath *string
	if p, ok := d.GetOk("nix_path"); ok {
		p := p.(string)
		nixPath = &p
	}

	expressionPath := d.Get("expression_path").(string)

	storePath, err := nix.BuildExpression(nixPath, expressionPath)
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
