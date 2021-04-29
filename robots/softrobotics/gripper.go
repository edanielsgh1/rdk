package softrobotics

import (
	"context"
	"fmt"
	"time"

	"go.viam.com/robotcore/api"
	"go.viam.com/robotcore/board"

	"github.com/edaniels/golog"
	"go.uber.org/multierr"
)

func init() {
	api.RegisterGripper("softrobotics", func(ctx context.Context, r api.Robot, config api.Component, logger golog.Logger) (api.Gripper, error) {
		b := r.BoardByName("local")
		if b == nil {
			return nil, fmt.Errorf("softrobotics gripper requires a board called local")
		}
		g, ok := b.(board.GPIOBoard)
		if !ok {
			return nil, fmt.Errorf("softrobotics gripper requires a baord that is a GPIOBoard")
		}
		return NewGripper(ctx, b, g, config, logger)
	})
}

/*
   open is 5
   close is 6
*/
type Gripper struct {
	theBoard  board.Board
	gpioBoard board.GPIOBoard

	psi board.AnalogReader

	pinOpen, pinClose, pinPower string

	logger golog.Logger
}

func NewGripper(ctx context.Context, b board.Board, g board.GPIOBoard, config api.Component, logger golog.Logger) (*Gripper, error) {
	theGripper := &Gripper{
		theBoard:  b,
		gpioBoard: g,
		psi:       b.AnalogReader("psi"),
		pinOpen:   config.Attributes.GetString("open"),
		pinClose:  config.Attributes.GetString("close"),
		pinPower:  config.Attributes.GetString("power"),
		logger:    logger,
	}

	if theGripper.psi == nil {
		return nil, fmt.Errorf("no psi analog reader")
	}

	if theGripper.pinOpen == "" || theGripper.pinClose == "" || theGripper.pinPower == "" {
		return nil, fmt.Errorf("need pins for open, close, power")
	}

	return theGripper, nil
}

func (g *Gripper) Stop() error {
	return multierr.Combine(
		g.gpioBoard.GPIOSet(g.pinOpen, false),
		g.gpioBoard.GPIOSet(g.pinClose, false),
		g.gpioBoard.GPIOSet(g.pinPower, false),
	)
}

func (g *Gripper) Open(ctx context.Context) error {
	err := multierr.Combine(
		g.gpioBoard.GPIOSet(g.pinOpen, true),
		g.gpioBoard.GPIOSet(g.pinPower, true),
	)
	if err != nil {
		return err
	}

	for {
		time.Sleep(time.Millisecond * 10) // REMOVE

		val, err := g.psi.Read(ctx)
		if err != nil {
			return multierr.Combine(err, g.Stop())
		}

		if val > 500 {
			break
		}

		time.Sleep(time.Millisecond * 10)
	}

	return g.Stop()
}

func (g *Gripper) Grab(ctx context.Context) (bool, error) {
	err := multierr.Combine(
		g.gpioBoard.GPIOSet(g.pinClose, true),
		g.gpioBoard.GPIOSet(g.pinPower, true),
	)
	if err != nil {
		return false, err
	}

	for {

		time.Sleep(time.Millisecond * 100) // REMOVE

		val, err := g.psi.Read(ctx)
		if err != nil {
			return false, multierr.Combine(err, g.Stop())
		}

		if val <= 200 {
			break
		}

		time.Sleep(time.Millisecond * 10)
	}

	return false, g.Stop()

}