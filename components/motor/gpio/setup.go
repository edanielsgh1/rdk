package gpio

import (
	"context"

	"github.com/edaniels/golog"
	"github.com/pkg/errors"

	"go.viam.com/rdk/components/board"
	"go.viam.com/rdk/components/encoder"
	"go.viam.com/rdk/components/motor"
	"go.viam.com/rdk/config"
	"go.viam.com/rdk/registry"
	"go.viam.com/rdk/utils"
)

// init registers a pi motor based on pigpio.
func init() {
	comp := registry.Component{
		Constructor: func(ctx context.Context, deps registry.Dependencies, config config.Component, logger golog.Logger) (interface{}, error) {
			actualBoard, motorConfig, err := getBoardFromRobotConfig(deps, config)
			if err != nil {
				return nil, err
			}

			m, err := NewMotor(actualBoard, *motorConfig, logger)
			if err != nil {
				return nil, err
			}

			if motorConfig.Encoder != "" {
				e, err := encoder.FromDependencies(deps, motorConfig.Encoder)
				if err != nil {
					return nil, err
				}

				m, err = WrapMotorWithEncoder(ctx, e, config, *motorConfig, m, logger)
				if err != nil {
					return nil, err
				}
			}

			err = m.Stop(ctx, nil)
			if err != nil {
				return nil, err
			}

			return m, nil
		},
	}

	registry.RegisterComponent(motor.Subtype, "gpio", comp)
	motor.RegisterConfigAttributeConverter("gpio")
}

func getBoardFromRobotConfig(deps registry.Dependencies, config config.Component) (board.Board, *motor.Config, error) {
	motorConfig, ok := config.ConvertedAttributes.(*motor.Config)
	if !ok {
		return nil, nil, utils.NewUnexpectedTypeError(motorConfig, config.ConvertedAttributes)
	}
	if motorConfig.BoardName == "" {
		return nil, nil, errors.New("expected board name in config for motor")
	}
	b, err := board.FromDependencies(deps, motorConfig.BoardName)
	if err != nil {
		return nil, nil, err
	}
	return b, motorConfig, nil
}