// https://docs.microsoft.com/en-us/windows/win32/xinput/getting-started-with-xinput

package main

import (
	"fmt"
	"time"
	"unsafe"
	"strings"
	"syscall"
	"math"
)

var xInput = syscall.NewLazyDLL("Xinput1_4.dll");
var xInputGetState = xInput.NewProc("XInputGetState");

var GamepadConnectedCallback     = func(int) {}
var GamepadDisconnectedCallback  = func(int) {}
var GamepadPollCallback          = func(int, XInputState) {}
var GamepadInputCallback         = func(int, XInputState) {}
var ConnectedPollTime    = time.Millisecond
var DisconnectedPollTime = time.Second

// These must be between 0.0 and 1.0
var ThumbstickDeadZone = DefaultThumbstickDeadZone
var TriggerThreshold   = DefaultTriggerThreshold

type XInputState struct {
	PacketNumber  DWORD
	Gamepad       XInputGamepad
}

type XInputGamepad struct {
	Buttons       WORD
	LeftTrigger   BYTE
	RightTrigger  BYTE
	ThumbLX       SHORT
	ThumbLY       SHORT
	ThumbRX       SHORT
	ThumbRY       SHORT
}

func (state XInputState) IsButtonDown(button WORD) bool {
	return state.Gamepad.Buttons & button != 0
}

// Return value is between 0.0 and 1.0
func (state XInputState) LeftTrigger() float32 {
	return trigger(state.Gamepad.LeftTrigger)
}

// Return value is between 0.0 and 1.0
func (state XInputState) RightTrigger() float32 {
	return trigger(state.Gamepad.RightTrigger)
}

func trigger(triggerValue BYTE) float32 {
	const MaxMagnitude = 255 // Max value of a BYTE ie max value of a trigger
	threshMagnitude := TriggerThreshold * MaxMagnitude
	if float64(triggerValue) > threshMagnitude {
		if triggerValue > MaxMagnitude {
			// Due to imperfect hardware this value can be exceeded
			triggerValue = MaxMagnitude
		}
		normTriggerValue := (float64(triggerValue) - threshMagnitude) / (MaxMagnitude - threshMagnitude)
		if normTriggerValue > 1 {
			normTriggerValue = 1
		}
		return float32(normTriggerValue)
	}
	return 0
}

// Return value is between 0.0 and 1.0
func (state XInputState) LeftThumbstickX() float32 {
	return state.thumbstick(state.Gamepad.ThumbLX, state.Gamepad.ThumbLY, true)
}

// Return value is between 0.0 and 1.0
func (state XInputState) LeftThumbstickY() float32 {
	return state.thumbstick(state.Gamepad.ThumbLX, state.Gamepad.ThumbLY, false)
}

// Return value is between 0.0 and 1.0
func (state XInputState) RightThumbstickX() float32 {
	return state.thumbstick(state.Gamepad.ThumbRX, state.Gamepad.ThumbRY, true)
}

// Return value is between 0.0 and 1.0
func (state XInputState) RightThumbstickY() float32 {
	return state.thumbstick(state.Gamepad.ThumbRX, state.Gamepad.ThumbRY, false)
}

func (state XInputState) thumbstick(thumbstickX SHORT, thumbstickY SHORT, xAxis bool) float32 {
	const MaxMagnitude = 32767 // Max value of a SHORT ie max value of a thumbstick
	zoneMagnitude := ThumbstickDeadZone * MaxMagnitude	

	// I used this method to implement dead zones and normalization.
	// https://docs.microsoft.com/en-us/windows/win32/xinput/getting-started-with-xinput#dead-zone
	x := float64(thumbstickX)
	y := float64(thumbstickY)
	magnitude := math.Sqrt(x*x + y*y)
	if magnitude > zoneMagnitude {
		if magnitude > MaxMagnitude {
			// Due to imperfect hardware this value can be exceeded
			magnitude = MaxMagnitude
		}
		// This is just a shortened version of what the article does
		// in the link above.
		var numerator float64
		if xAxis {
			numerator = x * (magnitude - zoneMagnitude)
		} else {
			numerator = y * (magnitude - zoneMagnitude)
		}
		denominator := magnitude * (MaxMagnitude - zoneMagnitude)
		// Normalized to 0.0 to 1.0
		normLx := float32(numerator / denominator)
		if normLx > 1 {
			normLx = 1
		} else if normLx < -1 {
			normLx = -1
		}
		return normLx
	}
	return 0
}

type GamepadInput struct {
	// Set only one of these three
    IsButton      bool
    IsTrigger     bool
    IsThumbstick  bool

    Button        WORD // Button code. See the constants prefixed by XINPUT_GAMEPAD_
	IsLeft        bool // Determines which trigger or thumbstick is used.
	IsX           bool // Determines the thumbstick axis.
}

func NewGamepadButtonInput(button WORD) GamepadInput {
	input := GamepadInput{}
	input.IsButton = true
	input.Button = button
	return input
}

func NewGamepadTriggerInput(isLeft bool) GamepadInput {
	input := GamepadInput{}
	input.IsTrigger = true
	input.IsLeft = true
	return input
}

func NewGamepadThumbstickInput(isLeft bool, isX bool) GamepadInput {
	input := GamepadInput{}
	input.IsThumbstick = true
	input.IsLeft = isLeft
	input.IsX = isX
	return input
}

/*
 * Return value depends in the input type:
 * if input.IsButton     returns 0 or 1
 * if input.IsTrigger    returns [0.0, 1.0]
 * if input.IsThumbstick returns [-1.0, 1.0]
 */
func (state XInputState) InputValueFloat(input GamepadInput) float32 {
    if input.IsButton {
        if state.IsButtonDown(input.Button) {
            return 1;
        } else {
            return 0;
        }
    } else if input.IsThumbstick {
		if input.IsLeft {
			if input.IsX {
				return state.LeftThumbstickX();
			} else {
				return state.LeftThumbstickY();
			}
		} else {
			if input.IsX {
				return state.RightThumbstickX();
			} else {
				return state.RightThumbstickY();
			}
		}
    } else if input.IsTrigger {
		if input.IsLeft {
			return state.LeftTrigger();
		} else {
			return state.RightTrigger();
		}
	} else {
		panic("GamepadInput struct had no input type.");
	}
}

// Returns true if the button is down or the trigger/thumbstick is non-zero.
func (state XInputState) InputValueBool(input GamepadInput) bool {
	val := state.InputValueFloat(input);
	if val == 0 {
		return false;
	} else {
		return true;
	}
}

// Intended usage: Set the callback functions and call this in a goroutine
func PollGamepad(userIndex int) {
	var state XInputState
	found := false
	connectCbCalled := false
	disconnectCbCalled := false
	var previousPacketNumber DWORD = 0
	
	for {
		state, found = getGamepadState(userIndex)
		if found {
			disconnectCbCalled = false
			if !connectCbCalled {
				GamepadConnectedCallback(userIndex)
				connectCbCalled = true
			}
			GamepadPollCallback(userIndex, state)
			if previousPacketNumber != state.PacketNumber {
				GamepadInputCallback(userIndex, state)
				previousPacketNumber = state.PacketNumber
			}
			time.Sleep(ConnectedPollTime)
		} else {
			connectCbCalled = false
			if !disconnectCbCalled {
				GamepadDisconnectedCallback(userIndex)
				disconnectCbCalled = true
			}
			time.Sleep(DisconnectedPollTime)
		}
	}
}

func (state XInputState) String() string {
	var s strings.Builder
	s.WriteString("XInputState {\n")
	s.WriteString(fmt.Sprintf("\tPacketNumber: %d\n", state.PacketNumber))
	s.WriteString("\tButtons: ")
	for key, value := range GamepadButtonToString {
		if state.IsButtonDown(WORD(key)) {
			s.WriteString(value + " ")
		}
	}
	s.WriteString(fmt.Sprintf("\n\tTriggers: %f %f\n", state.LeftTrigger(), state.RightTrigger()))
	s.WriteString(fmt.Sprintf("\tLeft thumbstick: %f %f\n", state.LeftThumbstickX(), state.LeftThumbstickY()))
	s.WriteString(fmt.Sprintf("\tRight thumbstick: %f %f\n", state.RightThumbstickX(), state.RightThumbstickY()))
	s.WriteString("}")
	return s.String()
}

func getGamepadState(userIndex int) (XInputState, bool) {
	state := XInputState{}
	// https://docs.microsoft.com/en-us/windows/win32/api/xinput/nf-xinput-xinputgetstate
	status, _, err := xInputGetState.Call(
		uintptr(userIndex), // 0-3
		uintptr(unsafe.Pointer(&state)),
	)
	panicIfSyscallErr(err)
	if status == ERROR_SUCCESS {
		return state, true
	} else if status == ERROR_DEVICE_NOT_CONNECTED {
		return state, false
	} else if status == ERROR_BAD_ARGUMENTS {
		panic("Internal error: bad arguments")
	} else {
		panic("Internal error: unexpected status")
	}
}

func panicIfSyscallErr(err error) {
	if err != syscall.Errno(0) {
		panic(err)
	}
}

func init() {
    GamepadButtonToString = map[int]string{}
    for k,v := range(StringToGamepadButton) {
        GamepadButtonToString[v] = k
    }
}

var StringToGamepadButton = map[string]int {
    // Dpad
	"UP"        : XINPUT_GAMEPAD_DPAD_UP,
    "DPADUP"    : XINPUT_GAMEPAD_DPAD_UP,
	"DOWN"      : XINPUT_GAMEPAD_DPAD_DOWN,
	"DPADDOWN"  : XINPUT_GAMEPAD_DPAD_DOWN,
	"LEFT"      : XINPUT_GAMEPAD_DPAD_LEFT,
	"DPADLEFT"  : XINPUT_GAMEPAD_DPAD_LEFT,
	"RIGHT"     : XINPUT_GAMEPAD_DPAD_RIGHT,
	"DPADRIGHT" : XINPUT_GAMEPAD_DPAD_RIGHT,

	// Thumbsticks
	"LTHUMBCLICK"         : XINPUT_GAMEPAD_LEFT_THUMB,
	"LSTICKCLICK"         : XINPUT_GAMEPAD_LEFT_THUMB,
	"LEFTTHUMBCLICK"      : XINPUT_GAMEPAD_LEFT_THUMB,
	"LEFTSTICKCLICK"      : XINPUT_GAMEPAD_LEFT_THUMB,
	"LEFTTHUMBSTICKCLICK" : XINPUT_GAMEPAD_LEFT_THUMB,

	"RTHUMBCLICK"          : XINPUT_GAMEPAD_RIGHT_THUMB,
	"RSTICKCLICK"          : XINPUT_GAMEPAD_RIGHT_THUMB,
	"RIGHTTHUMBCLICK"      : XINPUT_GAMEPAD_RIGHT_THUMB,
	"RIGHTSTICKCLICK"      : XINPUT_GAMEPAD_RIGHT_THUMB,
	"RIGHTTHUMBSTICKCLICK" : XINPUT_GAMEPAD_RIGHT_THUMB,

	// Shoulders
	"LBUMPER"      : XINPUT_GAMEPAD_LEFT_SHOULDER,
	"LEFTBUMPER"   : XINPUT_GAMEPAD_LEFT_SHOULDER,
	"LSHOULDER"    : XINPUT_GAMEPAD_LEFT_SHOULDER,
	"LEFTSHOULDER" : XINPUT_GAMEPAD_LEFT_SHOULDER,

	"RBUMPER"       : XINPUT_GAMEPAD_RIGHT_SHOULDER,
	"RIGHTBUMPER"   : XINPUT_GAMEPAD_RIGHT_SHOULDER,
	"RSHOULDER"     : XINPUT_GAMEPAD_RIGHT_SHOULDER,
	"RIGHTSHOULDER" : XINPUT_GAMEPAD_RIGHT_SHOULDER,

	// Action buttons
	"A"     : XINPUT_GAMEPAD_A,
	"B"     : XINPUT_GAMEPAD_B,
	"X"     : XINPUT_GAMEPAD_X,
	"Y"     : XINPUT_GAMEPAD_Y,
	"START" : XINPUT_GAMEPAD_START,
	"BACK"  : XINPUT_GAMEPAD_BACK,
}
// Initialized in init()
var GamepadButtonToString map[int]string

const (
	DefaultThumbstickDeadZone = 0.25
	DefaultTriggerThreshold   = 0.1

	// Return values from XInputGetState
	ERROR_SUCCESS              = 0
	ERROR_BAD_ARGUMENTS        = 160
	ERROR_DEVICE_NOT_CONNECTED = 1167

	// For use in XInputGamepad.Buttons
	XINPUT_GAMEPAD_DPAD_UP        = 0x0001
	XINPUT_GAMEPAD_DPAD_DOWN      = 0x0002
	XINPUT_GAMEPAD_DPAD_LEFT      = 0x0004
	XINPUT_GAMEPAD_DPAD_RIGHT     = 0x0008
	XINPUT_GAMEPAD_START          = 0x0010
	XINPUT_GAMEPAD_BACK           = 0x0020
	XINPUT_GAMEPAD_LEFT_THUMB     = 0x0040
	XINPUT_GAMEPAD_RIGHT_THUMB    = 0x0080
	XINPUT_GAMEPAD_LEFT_SHOULDER  = 0x0100
	XINPUT_GAMEPAD_RIGHT_SHOULDER = 0x0200
	XINPUT_GAMEPAD_A              = 0x1000
	XINPUT_GAMEPAD_B              = 0x2000
	XINPUT_GAMEPAD_X              = 0x4000
	XINPUT_GAMEPAD_Y              = 0x8000
)
