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
			"expression": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
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
	Expression     string
	ExpressionPath string
	NixPath        string
	OutLink        string
}

func (cfg *nixBuildResourceConfig) DoBuild() (string, error) {
	return cfg.doBuild(&cfg.OutLink)
}

func (cfg *nixBuildResourceConfig) DoBuildNoLink() (string, error) {
	return cfg.doBuild(nil)
}

func (cfg *nixBuildResourceConfig) doBuild(outLink *string) (string, error) {
	if cfg.Expression != "" {
		f, err := os.Create(cfg.ExpressionPath)
		if err != nil {
			return "", err
		}
		_, err = f.Write([]byte(cfg.Expression))
		if err != nil {
			return "", err
		}
		err = f.Close()
		if err != nil {
			return "", err
		}
	}

	return nix.BuildExpression(cfg.NixPath, cfg.ExpressionPath, outLink)
}

func getBuildConfig(d resourceLike) (nixBuildResourceConfig, error) {

	nixPath := os.Getenv("NIX_PATH")
	if p, ok := d.GetOk("nix_path"); ok {
		nixPath = p.(string)
	}

	expression, _ := d.GetOk("expression")

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
		Expression:     expression.(string),
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

	// Delete the old expression if it was under out control.
	if d.HasChange("expression_path") {
		oldExpression, _ := d.GetChange("expression")
		if oldExpression != "" {
			old, _ := d.GetChange("expression_path")
			err = os.Remove(old.(string))
			if err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}

	linkExists := false
	_, err = os.Readlink(cfg.OutLink)
	if err == nil {
		linkExists = true
	}

	if d.IsNewResource() || d.HasChange("store_path") || !linkExists {
		_, err = cfg.DoBuild()
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

	if cfg.Expression != "" {
		err = os.Remove(cfg.ExpressionPath)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
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
	// A trick to prevent prematurely writing nix expressions to disks path
	// when this is the first diff.
	if d.HasChange("expression") {
		d.SetNewComputed("store_path")
		return nil
	}

	cfg, err := getBuildConfig(d)
	if err != nil {
		return err
	}

	desiredBuild, err := cfg.DoBuildNoLink()
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
