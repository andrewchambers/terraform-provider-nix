package nix

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func formatChildErr(err error) error {
	if err == nil {
		return nil
	}

	if err, ok := err.(*exec.ExitError); ok {
		return errors.New(string(err.Stderr))
	}
	return err
}

// BuildExpression builds a nix expression, returning the store path.
func BuildExpression(nixPath *string, expressionPath string) (string, error) {

	tempDir, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tempDir)

	cmd := exec.Command("nix-build", "--no-link", expressionPath)

	if nixPath != nil {
		cmd.Env = []string{fmt.Sprintf("NIX_PATH=%s", *nixPath)}
	} else {
		cmd.Env = []string{fmt.Sprintf("NIX_PATH=%s", os.Getenv("NIX_PATH"))}
	}

	output, err := runCommandWithLogging(cmd)
	if err != nil {
		return "", fmt.Errorf("building expression failed: %s", formatChildErr(err))
	}

	return strings.TrimSpace(string(output)), nil
}

// NixosRebuildConfig represents a configuration for Nixos rebuild.
type NixosRebuildConfig struct {
	TargetHost  string
	TargetUser  string
	BuildHost   string
	NixosConfig string
	NixPath     string
	SSHOpts     string
}

// GetEnv returns an OS env suitable for nixos-rebuild.
func (cfg *NixosRebuildConfig) GetEnv() []string {
	env := os.Environ()
	env = append(env, fmt.Sprintf("NIX_PATH=%s", cfg.NixPath))
	env = append(env, fmt.Sprintf("NIX_SSHOPTS=%s", cfg.SSHOpts))
	env = append(env, fmt.Sprintf("NIXOS_CONFIG=%s", cfg.NixosConfig))
	return env
}

// WaitForSSH waits until the given ssh host is up and ready for commands.
func WaitForSSH(user, host, sshOpts string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	cmd := exec.Command("sh", "-c", fmt.Sprintf("exec ssh %s %s@%s -G", sshOpts, user, host))
	out, err := cmd.Output() // Not interested in this in the logs...
	if err != nil {
		return err
	}
	outs := string(out)

	host = ""
	port := ""

	lines := strings.Split(outs, "\n")
	for _, line := range lines {
		line := strings.TrimSpace(line)

		if strings.HasPrefix(line, "hostname") {
			host = line[9:]
		}

		if strings.HasPrefix(line, "port") {
			port = line[5:]
		}
	}

	for {
		if time.Now().After(deadline) {
			return errors.New("ssh server down or not responsive")
		}
		dialer := net.Dialer{
			Timeout: 10 * time.Second,
		}
		c, err := dialer.Dial("tcp", fmt.Sprintf("%s:%s", host, port))
		if err == nil {
			_ = c.Close()
			break
		}
		time.Sleep(2 * time.Second)
	}

	cmd = exec.Command("sh", "-c", fmt.Sprintf("exec timeout 10s ssh %s %s@%s -- true", sshOpts, user, host))
	_, err = runCommandWithLogging(cmd)
	if err != nil {
		return err
	}

	return nil
}

// BuildSystem builds a nixos system config and returns the store path.
func BuildSystem(cfg *NixosRebuildConfig) (string, error) {
	tmp, err := ioutil.TempDir("", "")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmp)

	outLink := filepath.Join(tmp, "result")
	outLink, err = filepath.Abs(outLink)
	if err != nil {
		return "", err
	}

	cmd := exec.Command("nixos-rebuild", "build", "--build-host", cfg.BuildHost)
	cmd.Dir = tmp
	cmd.Env = cfg.GetEnv()
	_, err = runCommandWithLogging(cmd)
	if err != nil {
		return "", formatChildErr(err)
	}

	return os.Readlink(outLink)
}

// CurrentSystem returns the store path of the system on the TargetHost.
func CurrentSystem(cfg *NixosRebuildConfig) (string, error) {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("exec timeout 10s ssh %s %s@%s -- readlink /run/current-system", cfg.SSHOpts, cfg.TargetUser, cfg.TargetHost))
	output, err := runCommandWithLogging(cmd)
	return strings.TrimSpace(string(output)), formatChildErr(err)
}

// SwitchSystem is the equivalent of nixos-rebuild switch.
func SwitchSystem(cfg *NixosRebuildConfig) error {
	cmd := exec.Command("nixos-rebuild", "switch", "--build-host", cfg.BuildHost, "--target-host", fmt.Sprintf("%s@%s", cfg.TargetUser, cfg.TargetHost))
	cmd.Env = cfg.GetEnv()
	_, err := runCommandWithLogging(cmd)
	return err
}

// CollectGarbage runs nix-collect-garbage -d on the remote host.
func CollectGarbage(user, host, sshOpts string) error {
	cmd := exec.Command("sh", "-c", fmt.Sprintf("exec ssh %s %s@%s -- nix-collect-garbage -d", sshOpts, user, host))
	_, err := runCommandWithLogging(cmd)
	return formatChildErr(err)
}
