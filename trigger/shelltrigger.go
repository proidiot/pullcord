package trigger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/proidiot/gone/log"
	"github.com/stuphlabs/pullcord/config"
)

// ShellTriggerrer is a basic Triggerrer that calls a stored shell
// command (along with arguments) when triggered.
//
// The message given to TriggerString will be passed to the command via stdin.
type ShellTriggerrer struct {
	Command string
	Args    []string
}

func init() {
	config.RegisterResourceType(
		"shelltrigger",
		func() json.Unmarshaler {
			return new(ShellTriggerrer)
		},
	)
}

// UnmarshalJSON implements encoding/json.Unmarshaler.
func (s *ShellTriggerrer) UnmarshalJSON(input []byte) error {
	// It shouldn't habe been necessary to do this, but by giving a
	// defnition of how to unmarshal a pointer-to ShellTriggerrer
	// (which we apparently need to do), it seems that unmarshalling a
	// non-pointer ShellTriggerrer also uses this function to
	// unmarshal, resulting in an infinite stack.
	var t struct {
		Command string
		Args    []string
	}

	dec := json.NewDecoder(bytes.NewReader(input))
	if e := dec.Decode(&t); e != nil {
		return e
	}

	s.Command = t.Command
	s.Args = t.Args

	return nil
}

// Trigger will execute the given command with the given args using the system
// shell.
func (s *ShellTriggerrer) Trigger() (err error) {
	log.Debug("shelltrigger running trigger")
	cmd := exec.Command(s.Command, s.Args...)
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	err = cmd.Run()
	log.Debug(
		fmt.Sprintf(
			"shelltrigger command wrote to stdout: %s",
			stdout.String(),
		),
	)
	if err != nil {
		log.Err(
			fmt.Sprintf(
				"shelltrigger failed during trigger: %v",
				err,
			),
		)
		return err
	}

	log.Info("shelltrigger trigger sent")
	return nil
}

// NewShellTriggerrer constructs a new ShellTriggerrer given the
// command (and arguments) to be run each time TriggerString is called. Entire
// shell scripts could potentially be stored in the arguments, though the
// trigger could just as easily call an external shell script. As a result, a
// wide variety of actions could be taken based on the message passed in via
// stdin.
func NewShellTriggerrer(
	command string,
	args []string,
) *ShellTriggerrer {
	log.Info("initializing shell trigger handler")
	log.Debug(
		fmt.Sprintf(
			"shelltrigger will run command: %s %v",
			command,
			args,
		),
	)

	var handler ShellTriggerrer
	handler.Command = command
	handler.Args = args

	return &handler
}
