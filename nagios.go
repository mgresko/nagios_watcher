package main

import "os/exec"

func NagiosTestConfig() ([]byte, error) {
	cmd := exec.Command("echo", "blah")
	output, err := cmd.CombinedOutput()
	return output, err
}

func NagiosRestart() ([]byte, error) {
	cmd := exec.Command("/etc/init.d/nagios3", "restart")
	output, err := cmd.CombinedOutput()
	return output, err
}
