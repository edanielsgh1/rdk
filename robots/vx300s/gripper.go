package vx300s

import (
	"context"
	"strconv"
	"sync"
	"time"

	"github.com/edaniels/golog"
	"github.com/jacobsa/go-serial/serial"

	"go.viam.com/dynamixel/network"
	"go.viam.com/dynamixel/servo"
	"go.viam.com/dynamixel/servo/s_model"

	"go.viam.com/robotcore/api"
)

func init() {
	api.RegisterGripper("vx300s", func(ctx context.Context, r api.Robot, config api.Component, logger golog.Logger) (api.Gripper, error) {
		return NewGripper(config.Attributes, getProviderOrCreate(r).moveLock, logger)
	})
}

type Gripper struct {
	jServo   *servo.Servo
	moveLock *sync.Mutex
}

func NewGripper(attributes api.AttributeMap, mutex *sync.Mutex, logger golog.Logger) (*Gripper, error) {
	jServo := findServo(attributes.GetString("usbPort"), attributes.GetString("baudRate"), logger)
	if mutex == nil {
		mutex = &sync.Mutex{}
	}
	err := jServo.SetTorqueEnable(true)
	newGripper := Gripper{
		jServo:   jServo,
		moveLock: mutex,
	}
	return &newGripper, err
}

func (g *Gripper) GetMoveLock() *sync.Mutex {
	return g.moveLock
}

func (g *Gripper) Open(ctx context.Context) error {
	g.moveLock.Lock()
	defer g.moveLock.Unlock()
	err := g.jServo.SetGoalPWM(150)
	if err != nil {
		return err
	}

	// We don't want to over-open
	atPos := false
	for !atPos {
		var pos int
		pos, err = g.jServo.PresentPosition()
		if err != nil {
			return err
		}
		if pos < 2800 {
			time.Sleep(50 * time.Millisecond)
		} else {
			atPos = true
		}
	}
	err = g.jServo.SetGoalPWM(0)
	return err
}

func (g *Gripper) Grab(ctx context.Context) (bool, error) {
	g.moveLock.Lock()
	defer g.moveLock.Unlock()
	err := g.jServo.SetGoalPWM(-350)
	if err != nil {
		return false, err
	}
	err = servo.WaitForMovementVar(g.jServo)
	if err != nil {
		return false, err
	}
	pos, err := g.jServo.PresentPosition()
	if err != nil {
		return false, err
	}
	didGrab := true

	// If servo position is less than 1500, it's closed and we grabbed nothing
	if pos < 1500 {
		didGrab = false
	}
	return didGrab, nil
}

// closes the connection, not the gripper
func (g *Gripper) Close() error {
	err := g.jServo.SetTorqueEnable(false)
	return err
}

// Find the gripper numbered Dynamixel servo on the specified USB port
// We're going to hardcode some USB parameters that we will literally never want to change
func findServo(usbPort, baudRateStr string, logger golog.Logger) *servo.Servo {
	GripperServoNum := 9
	baudRate, err := strconv.Atoi(baudRateStr)
	if err != nil {
		logger.Fatalf("Mangled baudrate: %v\n", err)
	}
	options := serial.OpenOptions{
		PortName:              usbPort,
		BaudRate:              uint(baudRate),
		DataBits:              8,
		StopBits:              1,
		MinimumReadSize:       0,
		InterCharacterTimeout: 100,
	}

	serial, err := serial.Open(options)
	if err != nil {
		logger.Fatalf("error opening serial port: %v\n", err)
	}

	network := network.New(serial)

	// By default, Dynamixel servos come 1-indexed out of the box because reasons
	//Get model ID of servo
	newServo, err := s_model.New(network, GripperServoNum)
	if err != nil {
		logger.Fatalf("error initializing servo %d: %v\n", GripperServoNum, err)
	}

	return newServo
}