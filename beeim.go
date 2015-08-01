package main

import (
	"beeim/s2c"
	//"fmt"
	//"os"
	//"os/exec"
)

func main() {
	/*if os.Getppid() != 1 {
		ss := exec.Command(os.Args[0], os.Args[1:]...)
		ss.Stdout = os.Stdout
		ss.Start()
		fmt.Println("[PID]", ss.Process.Pid)
		return
	}*/
	s2c.CreateServer(":1212")
}
