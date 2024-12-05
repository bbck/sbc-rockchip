package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/siderolabs/go-copy/copy"
	"github.com/siderolabs/talos/pkg/machinery/overlay"
	"github.com/siderolabs/talos/pkg/machinery/overlay/adapter"
	"golang.org/x/sys/unix"
)

const (
	off int64 = 512 * 64
	dtb       = "rockchip/rk3588-friendlyelec-cm3588-nas.dtb"
)

func main() {
	adapter.Execute(&CM3588Installer{})
}

type CM3588Installer struct{}

type CM3588ExtraOptions struct{}

func (i *CM3588Installer) GetOptions(extra CM3588ExtraOptions) (overlay.Options, error) {
	kernelArgs := []string{
		"console=tty0",
		"console=ttyS2,1500000",
		"sysctl.kernel.kexec_load_disabled=1",
		"talos.dashboard.disabled=1",
	}

	return overlay.Options{
		Name:       "friendlyelec-cm3588-nas",
		KernelArgs: kernelArgs,
		PartitionOptions: overlay.PartitionOptions{
			Offset: 2048 * 10,
		},
	}, nil
}

func (i *CM3588Installer) Install(options overlay.InstallOptions[CM3588ExtraOptions]) error {
	var err error

	uBootBin := filepath.Join(options.ArtifactsPath, "arm64/u-boot/friendlyelec-cm3588-nas/u-boot-rockchip.bin")

	err = uBootLoaderInstall(uBootBin, options.InstallDisk)
	if err != nil {
		return err
	}

	src := filepath.Join(options.ArtifactsPath, "arm64/dtb", dtb)
	dst := filepath.Join(options.MountPrefix, "boot/EFI/dtb", dtb)

	err = copyFileAndCreateDir(src, dst)
	if err != nil {
		return err
	}

	return nil
}

func copyFileAndCreateDir(src, dst string) error {
	err := os.MkdirAll(filepath.Dir(dst), 0o600)
	if err != nil {
		return err
	}

	return copy.File(src, dst)
}

func uBootLoaderInstall(uBootBin, installDisk string) error {
	var f *os.File

	f, err := os.OpenFile(installDisk, os.O_RDWR|unix.O_CLOEXEC, 0o666)
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", installDisk, err)
	}

	defer f.Close() //nolint:errcheck

	uboot, err := os.ReadFile(uBootBin)
	if err != nil {
		return err
	}

	if _, err = f.WriteAt(uboot, off); err != nil {
		return err
	}

	// NB: In the case that the block device is a loopback device, we sync here
	// to esure that the file is written before the loopback device is
	// unmounted.
	err = f.Sync()
	return err
}
