package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/andrewchambers/terraform-provider-nix/nix"
	"github.com/hashicorp/terraform/helper/schema"
)

// A NixBuild server somewhere in the ether.
func resourceNixBuild() *schema.Resource {
	return &schema.Resource{
		Create:        resourceNixBuildCreateUpdate,
		Update:        resourceNixBuildCreateUpdate,
		Read:          resourceNixBuildRead,
		Delete:        resourceNixBuildDelete,
		Exists:        resourceNixBuildExists,
		CustomizeDiff: resourceNixBuildCustomizeDiff,

		Schema: map[string]*schema.Schema{
			"expression_path": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"nix_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"store_path": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"out_link": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
		},
	}
}

type nixBuildResourceConfig struct {
	ExpressionPath string
	NixPath        string
	OutLink        string
}

func getBuildConfig(d resourceLike) (nixBuildResourceConfig, error) {

	nixPath := os.Getenv("NIX_PATH")
	if p, ok := d.GetOk("nix_path"); ok {
		nixPath = p.(string)
	}

	expressionPath := d.Get("expression_path").(string)
	expressionPath, err := filepath.Abs(expressionPath)
	if err != nil {
		return nixBuildResourceConfig{}, err
	}

	outLink := d.Get("out_link").(string)
	outLink, err = filepath.Abs(outLink)
	if err != nil {
		return nixBuildResourceConfig{}, err
	}

	return nixBuildResourceConfig{
		NixPath:        nixPath,
		ExpressionPath: expressionPath,
		OutLink:        outLink,
	}, nil
}

func resourceNixBuildCreateUpdate(d *schema.ResourceData, m interface{}) error {

	id := d.Id()
	if id == "" {
		d.SetId(randomID())
	}

	cfg, err := getBuildConfig(d)
	if err != nil {
		return err
	}

	if d.HasChange("out_link") {
		old, _ := d.GetChange("out_link")
		err = os.Remove(old.(string))
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	linkExists := false
	_, err = os.Readlink(cfg.OutLink)
	if err == nil {
		linkExists = true
	}

	if d.IsNewResource() || d.HasChange("store_path") || !linkExists {
		_, err = nix.BuildExpression(cfg.NixPath, cfg.ExpressionPath, &cfg.OutLink)
		if err != nil {
			return err
		}
	}

	return resourceNixBuildRead(d, m)
}

func resourceNixBuildRead(d *schema.ResourceData, m interface{}) error {

	cfg, err := getBuildConfig(d)
	if err != nil {
		return err
	}

	storePath, err := os.Readlink(cfg.OutLink)
	if err != nil {
		return err
	}

	err = d.Set("store_path", storePath)
	if err != nil {
		return err
	}

	return nil
}

func resourceNixBuildDelete(d *schema.ResourceData, m interface{}) error {
	cfg, err := getBuildConfig(d)
	if err != nil {
		return err
	}

	err = os.Remove(cfg.OutLink)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}

func resourceNixBuildExists(d *schema.ResourceData, m interface{}) (bool, error) {
	cfg, err := getBuildConfig(d)
	if err != nil {
		return false, err
	}

	_, err = os.Readlink(cfg.OutLink)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

func resourceNixBuildCustomizeDiff(d *schema.ResourceDiff, m interface{}) error {
	cfg, err := getBuildConfig(d)
	if err != nil {
		return err
	}

	desiredBuild, err := nix.BuildExpression(cfg.NixPath, cfg.ExpressionPath, nil)
	if err != nil {
		log.Printf("build failed, assuming this is because of generated expression. err=%s", err.Error())
		d.SetNewComputed("store_path")
	} else {
		if d.Get("store_path").(string) != desiredBuild {
			d.SetNewComputed("store_path")
		}
	}

	return nil
}
