package main

import "os/exec"

func NagiosTestConfig() ([]byte, error) {
	cmd := exec.Command(*init_file, "check")
	output, err := cmd.CombinedOutput()
	return output, err
}

func NagiosRestart() ([]byte, error) {
	cmd := exec.Command(*init_file, "restart")
	output, err := cmd.CombinedOutput()
	return output, err
}
