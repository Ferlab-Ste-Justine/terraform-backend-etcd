package main

import (
	"os"
	"os/exec"
	"testing"
)

func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return true, err
		}

		return false, nil
	}

	return true, nil
}

func TerraformApply(orchPath string, t *testing.T) error {
	initCmd := exec.Command(
		"terraform",
		"init",
	)
	initCmd.Dir = orchPath
	initCmd.Stdout = os.Stdout
	initCmd.Stderr = os.Stderr
	cmdErr := initCmd.Run()
	if cmdErr != nil {
		return cmdErr
	}

	applyCmd := exec.Command(
		"terraform",
		"apply",
		"-auto-approve",
	)
	applyCmd.Dir = orchPath
	applyCmd.Stdout = os.Stdout
	applyCmd.Stderr = os.Stderr
	cmdErr = applyCmd.Run()
	if cmdErr != nil {
		return cmdErr
	}

	return nil
}

func TerraformDestroy(orchPath string, t *testing.T) error {
	initCmd := exec.Command(
		"terraform",
		"init",
	)
	initCmd.Dir = orchPath
	initCmd.Stdout = os.Stdout
	initCmd.Stderr = os.Stderr
	cmdErr := initCmd.Run()
	if cmdErr != nil {
		return cmdErr
	}

	applyCmd := exec.Command(
		"terraform",
		"apply",
		"-destroy",
		"-auto-approve",
	)
	applyCmd.Dir = orchPath
	applyCmd.Stdout = os.Stdout
	applyCmd.Stderr = os.Stderr
	cmdErr = applyCmd.Run()
	if cmdErr != nil {
		return cmdErr
	}

	return nil
}