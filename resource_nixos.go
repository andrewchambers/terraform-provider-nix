package main

import (
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/andrewchambers/terraform-provider-nix/nix"
	"github.com/hashicorp/terraform/helper/schema"
)

// A nixos server somewhere in the ether.
func resourceNixOS() *schema.Resource {
	return &schema.Resource{
		Create:        resourceNixOSCreateUpdate,
		Update:        resourceNixOSCreateUpdate,
		Read:          resourceNixOSRead,
		Delete:        resourceNixOSDelete,
		CustomizeDiff: resourceNixOSCustomizeDiff,

		Schema: map[string]*schema.Schema{
			"target_host": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"target_user": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "root",
			},
			"build_host": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "localhost",
			},
			"nixos_config": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"nixos_config_path": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"ssh_opts": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "-o StrictHostKeyChecking=accept-new -o BatchMode=yes",
			},
			"nix_path": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"ssh_timeout": &schema.Schema{
				Type:     schema.TypeInt,
				Optional: true,
				Default:  180,
			},
			"collect_garbage": &schema.Schema{
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
			"nixos_system": &schema.Schema{
				Type:     schema.TypeString,
				Computed: true,
			},
			"pre_switch_hook": &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				Default:   "",
				Sensitive: true,
			},
			"post_switch_hook": &schema.Schema{
				Type:      schema.TypeString,
				Optional:  true,
				Default:   "",
				Sensitive: true,
			},
		},
	}
}

type nixosResourceConfig struct {
	TargetHost      string
	TargetUser      string
	BuildHost       string
	NixosConfig     string
	NixosConfigPath string
	CollectGarbage  bool
	NixPath         string
	SSHOpts         string
	PreSwitchHook   string
	PostSwitchHook  string
	SSHTimeout      time.Duration
}

func (cfg *nixosResourceConfig) GetRebuildConfig() *nix.NixosRebuildConfig {
	return &nix.NixosRebuildConfig{
		TargetHost:      cfg.TargetHost,
		TargetUser:      cfg.TargetUser,
		BuildHost:       cfg.BuildHost,
		NixosConfigPath: cfg.NixosConfigPath,
		NixPath:         cfg.NixPath,
		SSHOpts:         cfg.SSHOpts,
		PreSwitchHook:   cfg.PreSwitchHook,
		PostSwitchHook:  cfg.PostSwitchHook,
	}
}

func (cfg *nixosResourceConfig) writeConfig() error {
	if cfg.NixosConfig != "" {
		f, err := os.Create(cfg.NixosConfigPath)
		if err != nil {
			return err
		}
		_, err = f.Write([]byte(cfg.NixosConfig))
		if err != nil {
			return err
		}
		err = f.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func (cfg *nixosResourceConfig) DoBuild() (string, error) {
	err := cfg.writeConfig()
	if err != nil {
		return "", err
	}

	return nix.BuildSystem(cfg.GetRebuildConfig())
}

func (cfg *nixosResourceConfig) DoSwitch() error {
	err := cfg.writeConfig()
	if err != nil {
		return err
	}

	return nix.SwitchSystem(cfg.GetRebuildConfig())
}

func (cfg *nixosResourceConfig) CurrentSystem() (string, error) {
	return nix.CurrentSystem(cfg.GetRebuildConfig())
}

func getNixosConfig(d resourceLike) (nixosResourceConfig, error) {

	nixPath, ok := d.GetOk("nix_path")
	if !ok {
		nixPath = os.Getenv("NIX_PATH")
	}

	sshOpts, ok := d.GetOk("ssh_opts")
	if !ok {
		sshOpts = os.Getenv("NIX_SSHOPTS")
	}

	nixosConfig, _ := d.GetOk("nixos_config")

	nixosConfigPath := d.Get("nixos_config_path").(string)

	nixosConfigPath, err := filepath.Abs(nixosConfigPath)
	if err != nil {
		return nixosResourceConfig{}, err
	}

	return nixosResourceConfig{
		TargetHost:      d.Get("target_host").(string),
		TargetUser:      d.Get("target_user").(string),
		BuildHost:       d.Get("build_host").(string),
		PreSwitchHook:   d.Get("pre_switch_hook").(string),
		PostSwitchHook:  d.Get("post_switch_hook").(string),
		NixosConfig:     nixosConfig.(string),
		NixosConfigPath: nixosConfigPath,
		NixPath:         nixPath.(string),
		SSHOpts:         sshOpts.(string),
		SSHTimeout:      time.Duration(d.Get("ssh_timeout").(int)) * time.Second,
		CollectGarbage:  d.Get("collect_garbage").(bool),
	}, nil
}

func resourceNixOSCreateUpdate(d *schema.ResourceData, m interface{}) error {

	id := d.Id()
	if id == "" {
		d.SetId(randomID())
	}

	cfg, err := getNixosConfig(d)
	if err != nil {
		return err
	}

	// Delete the old config if it was under out control.
	if d.HasChange("nixos_config_path") {
		oldConfig, _ := d.GetChange("nixos_config")
		if oldConfig != "" {
			old, _ := d.GetChange("nixos_config_path")
			err := os.Remove(old.(string))
			if err != nil && !os.IsNotExist(err) {
				return err
			}
		}
	}

	err = nix.WaitForSSH(cfg.TargetUser, cfg.TargetHost, cfg.SSHOpts, cfg.SSHTimeout)
	if err != nil {
		return err
	}

	if cfg.CollectGarbage {
		err = nix.CollectGarbage(cfg.TargetUser, cfg.TargetHost, cfg.SSHOpts)
		if err != nil {
			return err
		}
	}

	if d.HasChange("nixos_system") || d.HasChange("target_host") || d.HasChange("pre_switch_hook") || d.HasChange("post_switch_hook") {
		err = cfg.DoSwitch()
		if err != nil {
			return err
		}
	}

	return resourceNixOSRead(d, m)
}

func resourceNixOSRead(d *schema.ResourceData, m interface{}) error {

	cfg, err := getNixosConfig(d)
	if err != nil {
		return err
	}

	currentSystem := "unknown"

	err = nix.WaitForSSH(cfg.TargetUser, cfg.TargetHost, cfg.SSHOpts, cfg.SSHTimeout)
	if err == nil {
		currentSystem, err = cfg.CurrentSystem()
		if err != nil {
			return err
		}
	}

	err = d.Set("nixos_system", currentSystem)
	if err != nil {
		return err
	}

	return nil
}

func resourceNixOSDelete(d *schema.ResourceData, m interface{}) error {

	cfg, err := getNixosConfig(d)
	if err != nil {
		return err
	}

	if cfg.NixosConfig != "" {
		err := os.Remove(cfg.NixosConfigPath)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	return nil
}

func resourceNixOSCustomizeDiff(d *schema.ResourceDiff, m interface{}) error {
	// A trick to prevent prematurely writing nix expressions to disks path
	// when this is the first diff.
	if d.HasChange("nixos_config") {
		d.SetNewComputed("nixos_system")
		return nil
	}

	cfg, err := getNixosConfig(d)
	if err != nil {
		return err
	}

	desiredSystem, err := cfg.DoBuild()
	if err != nil {
		log.Printf("build failed, assuming this is because of generated configs. err=%s", err.Error())
		// If this really is an error, it will be picked up by the switch command.
		d.SetNewComputed("nixos_system")
		return nil
	}

	if d.Get("nixos_system").(string) != desiredSystem {
		d.SetNewComputed("nixos_system")
	}

	return nil
}
