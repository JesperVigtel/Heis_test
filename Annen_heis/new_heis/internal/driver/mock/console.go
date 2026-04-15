package mock

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"new_heis/internal/model"
)

func StartConsole(ctx context.Context, drv *Driver, in io.Reader, out io.Writer) {
	go func() {
		scan := bufio.NewScanner(in)
		help(out)
		for scan.Scan() {
			if ctx.Err() != nil {
				return
			}
			fields := strings.Fields(scan.Text())
			if len(fields) == 0 {
				continue
			}
			if err := runCommand(drv, fields, out); err != nil {
				fmt.Fprintf(out, "mock command error: %v\n", err)
			}
		}
	}()
}

func runCommand(drv *Driver, fields []string, out io.Writer) error {
	switch fields[0] {
	case "help":
		help(out)
	case "cab":
		floor, err := floorArg(fields)
		if err != nil {
			return err
		}
		drv.Press(model.ButtonCab, floor)
	case "hall-up":
		floor, err := floorArg(fields)
		if err != nil {
			return err
		}
		drv.Press(model.ButtonHallUp, floor)
	case "hall-down":
		floor, err := floorArg(fields)
		if err != nil {
			return err
		}
		drv.Press(model.ButtonHallDown, floor)
	case "obstruction":
		if len(fields) != 2 {
			return fmt.Errorf("usage: obstruction on|off")
		}
		if fields[1] != "on" && fields[1] != "off" {
			return fmt.Errorf("usage: obstruction on|off")
		}
		drv.SetObstruction(fields[1] == "on")
	case "status":
		s := drv.Snapshot()
		fmt.Fprintf(out, "floor=%d motor=%s doorOpen=%v lamps=%v\n", s.Floor, s.Motor, s.DoorOpen, s.ButtonLamps)
	default:
		return fmt.Errorf("unknown command %q", fields[0])
	}
	return nil
}

func floorArg(fields []string) (int, error) {
	if len(fields) != 2 {
		return 0, fmt.Errorf("usage: <command> <floor>")
	}
	return strconv.Atoi(fields[1])
}

func help(out io.Writer) {
	fmt.Fprintln(out, "mock commands: cab <floor>, hall-up <floor>, hall-down <floor>, obstruction on|off, status, help")
}
