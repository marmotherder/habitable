package command

import (
	"io"
	"os/exec"
	"sync"

	"github.com/marmotherder/habitable/common"
)

func RunCommand(directory string, command string, args ...string) error {
	common.AppLogger.Trace("running '%s %s' on host at %s", command, args, directory)
	cmd := exec.Command(command, args...)
	cmd.Dir = directory

	stdOut, err := cmd.StdoutPipe()
	if err != nil {
		common.AppLogger.Error("failed to open stdout for '%s %s' command", command, args)
		return err
	}
	stdErr, err := cmd.StderrPipe()
	if err != nil {
		common.AppLogger.Error("failed to open stderr for '%s %s' command", command, args)
		return err
	}
	if err := cmd.Start(); err != nil {
		common.AppLogger.Error("npm install failed to start")
		return err
	}

	var wg sync.WaitGroup
	wg.Add(2)
	processStream := func(stream io.ReadCloser) {
		defer wg.Done()
		buf := make([]byte, 80)
		for {
			n, err := stream.Read(buf)
			if n > 0 {
				common.AppLogger.Debug(string(buf[0:n]))
			}
			if err != nil {
				break
			}
		}
	}

	go processStream(stdOut)
	go processStream(stdErr)

	wg.Wait()

	return cmd.Wait()
}
