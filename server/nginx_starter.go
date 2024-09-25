package server

import (
	"fmt"
	"os/exec"
)

func StartOpenResty() error {
	cmd := exec.Command("openresty")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start OpenResty: %s", err)
	}
	return nil
}

func StopOpenResty() error {
	cmd := exec.Command("pkill", "openresty")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop OpenResty: %s", err)
	}
	return nil
}

// func StartNginx() error {
// 	cmd := exec.Command("sudo", "systemctl", "start", "nginx")
// 	err := cmd.Run()
// 	if err != nil {
// 		return fmt.Errorf("failed to start nginx: %v", err)
// 	}
// 	return nil
// }

// func StopNginx() error {
// 	cmd := exec.Command("sudo", "systemctl", "stop", "nginx")
// 	err := cmd.Run()
// 	if err != nil {
// 		return fmt.Errorf("failed to stop nginx: %v", err)
// 	}
// 	return nil
// }
