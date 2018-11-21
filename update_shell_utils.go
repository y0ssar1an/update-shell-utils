// update-shell-utils runs the following commands in parallel:
//   brew update && brew upgrade && brew cleanup && brew prune
//   pip3 install --upgrade && pip2 install --upgrade
//   go get -u <path>
//   rustup update
//   softwareupdate -ia
//   subl --command update_check
package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
	const numWorkers = 6
	errc := make(chan error, numWorkers)

	go func() {
		errc <- brewUpgrade()
	}()

	go func() {
		errc <- macOSUpdate()
	}()

	go func() {
		errc <- nvimPlugUpdate()
	}()

	go func() {
		errc <- pipUpgrade()
	}()

	go func() {
		errc <- rustupUpdate()
	}()

	go func() {
		errc <- sublPkgUpgrade()
	}()

	for i := 0; i < numWorkers; i++ {
		if err := <-errc; err != nil {
			fmt.Println(err)
		}
	}
}

func brewUpgrade() error {
	for _, subcmd := range []string{"update", "upgrade", "cleanup", "prune"} {
		if err := run("brew", subcmd); err != nil {
			return err
		}
	}

	return nil
}

func pipUpgrade() error {
	pkgs, err := outdatedPipPkgs()
	if err != nil {
		return err
	}
	if len(pkgs) == 0 {
		return nil
	}

	args := append([]string{"sudo", "-H", "pip3", "install", "--upgrade"}, pkgs...)
	return run("sudo", args...)
}

func outdatedPipPkgs() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sudo", "-H", "pip3", "list", "--format=freeze")
	var buf strings.Builder
	cmd.Stdout = &buf

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	return extractPipPkgs(buf.String()), nil
}

func extractPipPkgs(output string) []string {
	lines := strings.Split(output, "\n")
	pkgs := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		chunks := strings.Split(line, "==")
		if len(chunks) < 2 {
			continue // line doesn't contain "=="
		}

		pkg := strings.TrimSpace(chunks[0])
		pkgs = append(pkgs, pkg)
	}

	return pkgs
}

func rustupUpdate() error {
	err := run("rustup", "self", "update")
	if err != nil {
		return err
	}

	return run("rustup", "update")
}

func sublPkgUpgrade() error {
	err := run("subl", "--command", "update_check")
	if err != nil {
		return err
	}

	return run("subl", "--command", "upgrade_all_packages")
}

func nvimPlugUpdate() error {
	return run("nvim", "+PlugUpgrade", "+PlugUpdate", "+qa")
}

func macOSUpdate() error {
	// -i install updates
	// -a install *all* updates
	return run("sudo", "softwareupdate", "-ia")
}

func run(cmd string, args ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	command := exec.CommandContext(ctx, cmd, args...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr

	return command.Run()
}
