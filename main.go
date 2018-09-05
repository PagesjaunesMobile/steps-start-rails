package main

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/bitrise-io/go-utils/command"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-tools/go-steputils/tools"
	"github.com/dt-tools/servermanager"
)

// ConfigsModel ...
type ConfigsModel struct {
	WaitForBoot string
	BootTimeout string
}

func createConfigsModelFromEnvs() ConfigsModel {
	return ConfigsModel{
		WaitForBoot: os.Getenv("wait_for_boot"),
		BootTimeout: os.Getenv("boot_timeout"),
	}
}

func (configs ConfigsModel) print() {
	log.Infof("Configs:")
	log.Printf("- WaitForBoot: %s", configs.WaitForBoot)
	log.Printf("- BootTimeout: %s", configs.BootTimeout)
}

func (configs ConfigsModel) validate() error {
	if configs.WaitForBoot == "" {
		return errors.New("no WaitForBoot parameter specified")
	}
	if configs.BootTimeout == "" {
		return errors.New("no BootTimeout parameter specified")
	}

	return nil
}

func failf(format string, v ...interface{}) {
	log.Errorf(format, v...)
	os.Exit(1)
}

func main() {
	configs := createConfigsModelFromEnvs()

	fmt.Println()
	configs.print()

	if err := configs.validate(); err != nil {
		failf("Issue with input: %s", err)
	}

	// ---

	server, err := servermanager.New()
	if err != nil {
		failf("Failed to create server model, error: %s", err)
	}

	//
	// Start AVD image
	fmt.Println()
	log.Infof("Start Server")

	options := []string{}

	startServerCommand := server.StartServerCommand(options...)
	startServerCmd := startServerCommand.GetCmd()

	e := make(chan error)

	// Redirect output
	stdoutReader, err := startServerCmd.StdoutPipe()
	if err != nil {
		failf("Failed to redirect output, error: %s", err)
	}

	outScanner := bufio.NewScanner(stdoutReader)
	go func() {
		for outScanner.Scan() {
			line := outScanner.Text()
			fmt.Println(line)
		}
	}()
	if err := outScanner.Err(); err != nil {
		failf("Scanner failed, error: %s", err)
	}

	// Redirect error
	stderrReader, err := startServerCmd.StderrPipe()
	if err != nil {
		failf("Failed to redirect error, error: %s", err)
	}

	errScanner := bufio.NewScanner(stderrReader)
	go func() {
		for errScanner.Scan() {
			line := errScanner.Text()
			log.Warnf(line)
		}
	}()
	if err := errScanner.Err(); err != nil {
		failf("Scanner failed, error: %s", err)
	}
	// ---

	go func() {
		// Start emulator
		log.Printf("$ %s", command.PrintableCommandArgs(false, startServerCmd.Args))
		fmt.Println()

		if err := startServerCmd.Run(); err != nil {
			e <- err
			return
		}
	}()

	go func() {
		// Wait until device is booted
		if configs.WaitForBoot == "true" {
			bootInProgress := true
			for bootInProgress {
				time.Sleep(5 * time.Second)

				log.Printf("> Checking if server booted...")

				booted, err := server.IsBooted()
				if err != nil {
					e <- err
					return
				}

				bootInProgress = !booted
			}

			log.Donef("> server booted")
		}
		e <- nil
	}()

	timeout, err := strconv.ParseInt(configs.BootTimeout, 10, 64)
	if err != nil {
		failf("Failed to parse BootTimeout parameter, error: %s", err)
	}

	select {
	case <-time.After(time.Duration(timeout) * time.Second):
		if err := startServerCmd.Process.Kill(); err != nil {
			failf("Failed to kill emulator command, error: %s", err)
		}

		failf("Start server timed out")
	case err := <-e:
		if err != nil {
			failf("Failed to start emultor, error: %s", err)
		}

	}
	// ---

	if err := tools.ExportEnvironmentWithEnvman("BITRISE_SERVER_READY", "true"); err != nil {
		log.Warnf("Failed to export environment (BITRISE_SERVER_READY), error: %s", err)
	}

	fmt.Println()
	log.Donef("Server booted")
}
